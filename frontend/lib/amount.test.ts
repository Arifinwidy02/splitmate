import { describe, expect, it } from "vitest";

import {
  formatAmountInput,
  nextAmountInputValue,
  parseAmountInput,
} from "@/lib/amount";

describe("parseAmountInput", () => {
  it("strips non-numeric characters", () => {
    expect(parseAmountInput("abc")).toBe("");
    expect(parseAmountInput("12ab34")).toBe("1234");
    expect(parseAmountInput("Rp 50.000!")).toBe("50000");
  });

  it("treats a trailing separator with up to two digits as decimal", () => {
    expect(parseAmountInput("1000.5")).toBe("1000.5");
    expect(parseAmountInput("12,50")).toBe("12.50");
    expect(parseAmountInput("1.000,50")).toBe("1000.50");
  });

  it("treats separators followed by three or more digits as grouping", () => {
    expect(parseAmountInput("50.000")).toBe("50000");
    expect(parseAmountInput("1.500.000")).toBe("1500000");
    expect(parseAmountInput("1,500,000")).toBe("1500000");
  });

  it("treats a long trailing tail as grouping, not decimals", () => {
    expect(parseAmountInput("12.3456")).toBe("123456");
  });

  it("trims leading zeros", () => {
    expect(parseAmountInput("0003")).toBe("3");
    expect(parseAmountInput("0")).toBe("0");
  });
});

describe("formatAmountInput", () => {
  it("groups thousands with commas for both locales", () => {
    expect(formatAmountInput("1000")).toBe("1,000");
    expect(formatAmountInput("1500000")).toBe("1,500,000");
    expect(formatAmountInput("1000")).toBe("1,000");
    expect(formatAmountInput("1500000")).toBe("1,500,000");
  });

  it("formats decimals with a dot separator", () => {
    expect(formatAmountInput("1000.50")).toBe("1,000.50");
    expect(formatAmountInput("1000.50")).toBe("1,000.50");
  });

  it("returns an empty string for empty input", () => {
    expect(formatAmountInput("")).toBe("");
    expect(formatAmountInput("")).toBe("");
  });
});

describe("nextAmountInputValue", () => {
  it("appends digits after formatted grouping without mangling", () => {
    let parsed = "";
    let display = "";
    for (const ch of "150000") {
      parsed = nextAmountInputValue(parsed, display, display + ch);
      display = formatAmountInput(parsed);
    }
    expect(parsed).toBe("150000");
    expect(display).toBe("150,000");
  });

  it("appends a decimal separator and digits", () => {
    let parsed = "";
    let display = "";
    for (const ch of "1000,50") {
      parsed = nextAmountInputValue(parsed, display, display + ch);
      display = formatAmountInput(parsed);
    }
    expect(parsed).toBe("1000.50");
    expect(display).toBe("1,000.50");
  });

  it("ignores extra digits beyond two decimals", () => {
    const parsed = "1.50";
    const display = "1.50";
    expect(nextAmountInputValue(parsed, display, "1.500")).toBe("1.50");
  });

  it("backspaces the last character", () => {
    const parsed = "1000.5";
    const display = "1,000.5";
    expect(nextAmountInputValue(parsed, display, "1,000.")).toBe("1000.");
    expect(nextAmountInputValue("1000.", "1,000.", "1,000")).toBe("1000");
  });

  it("falls back to a full re-parse for mid-string edits", () => {
    expect(nextAmountInputValue("150000", "150,000", "1x50,000")).toBe("150000");
  });
});