import { describe, expect, it } from "vitest";

import { formatCurrency, formatDate, formatDateTime, formatSignedCurrency, toRFC3339 } from "./format";

describe("formatCurrency", () => {
  it("formats IDR with Rp prefix and id-ID grouping", () => {
    expect(formatCurrency("750000.00")).toBe("Rp750.000");
  });

  it("keeps two decimals when the amount is fractional", () => {
    expect(formatCurrency("123.45")).toBe("Rp123,45");
  });

  it("handles negative amounts", () => {
    expect(formatCurrency("-100000.00")).toBe("-Rp100.000");
  });

  it("formats zero", () => {
    expect(formatCurrency("0.00")).toBe("Rp0");
  });

  it("supports other currencies", () => {
    expect(formatCurrency("50.00", "USD")).toBe("USD 50");
  });

  it("returns the raw string for non-numeric input", () => {
    expect(formatCurrency("abc")).toBe("abc");
  });
});

describe("formatSignedCurrency", () => {
  it("prefixes positive balances with +", () => {
    expect(formatSignedCurrency("100000.00")).toBe("+Rp100.000");
  });

  it("prefixes negative balances with -", () => {
    expect(formatSignedCurrency("-100000.00")).toBe("-Rp100.000");
  });

  it("does not sign zero", () => {
    expect(formatSignedCurrency("0.00")).toBe("Rp0");
  });
});

describe("formatDate / formatDateTime", () => {
  it("formats dates in id-ID", () => {
    expect(formatDate("2026-08-14T19:00:00+07:00")).toBe("14 Agu 2026");
  });

  it("formats date and time", () => {
    expect(formatDateTime("2026-08-14T19:00:00+07:00")).toBe("14 Agu 2026, 19.00");
  });
});

describe("toRFC3339", () => {
  it("converts datetime-local values to RFC3339 with local offset", () => {
    const tzOffset = -new Date().getTimezoneOffset();
    const sign = tzOffset >= 0 ? "+" : "-";
    const abs = Math.abs(tzOffset);
    const offset = `${sign}${String(Math.floor(abs / 60)).padStart(2, "0")}:${String(
      abs % 60,
    ).padStart(2, "0")}`;

    expect(toRFC3339("2026-08-14T12:34")).toBe(`2026-08-14T12:34:00${offset}`);
  });

  it("passes through values that are already RFC3339", () => {
    expect(toRFC3339("2026-08-14T12:34:00+07:00")).toBe("2026-08-14T12:34:00+07:00");
  });

  it("passes through invalid values unchanged", () => {
    expect(toRFC3339("not-a-date")).toBe("not-a-date");
  });
});
