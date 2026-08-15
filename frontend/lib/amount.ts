import type { Locale } from "@/lib/i18n/constants";

export function parseAmountInput(raw: string): string {
  const cleaned = raw.replace(/[^\d.,]/g, "");

  const separators: number[] = [];
  for (let i = 0; i < cleaned.length; i++) {
    if (cleaned[i] === "." || cleaned[i] === ",") separators.push(i);
  }

  let decIndex = -1;
  if (separators.length > 0) {
    const last = separators[separators.length - 1];
    const trailing = cleaned.slice(last + 1).replace(/[.,]/g, "");
    if (trailing.length <= 2) decIndex = last;
  }

  const intPart = (
    decIndex === -1 ? cleaned : cleaned.slice(0, decIndex)
  ).replace(/[.,]/g, "");
  const decPart =
    decIndex === -1
      ? ""
      : cleaned.slice(decIndex + 1).replace(/[.,]/g, "").slice(0, 2);

  const result = decPart ? `${intPart}.${decPart}` : intPart;
  return result.replace(/^0+(?=\d)/, "");
}

export function formatAmountInput(parsed: string, locale: Locale): string {
  if (!parsed) return "";
  const [intPart, decPart] = parsed.split(".");
  const groupLocale = locale === "id" ? "id-ID" : "en-US";
  const grouped = new Intl.NumberFormat(groupLocale).format(Number(intPart || "0"));
  if (decPart === undefined) return grouped;
  const sep = locale === "id" ? "," : ".";
  return `${grouped}${sep}${decPart}`;
}

export function nextAmountInputValue(
  prevParsed: string,
  prevDisplay: string,
  newValue: string,
): string {
  if (newValue.startsWith(prevDisplay) && newValue.length === prevDisplay.length + 1) {
    const ch = newValue[newValue.length - 1];
    if (/[0-9]/.test(ch)) {
      const [, dec] = prevParsed.split(".");
      if (dec === undefined || dec.length < 2) return prevParsed + ch;
      return prevParsed;
    }
    if ((ch === "." || ch === ",") && !prevParsed.includes(".")) {
      return `${prevParsed}.`;
    }
    return prevParsed;
  }

  if (prevDisplay.startsWith(newValue) && prevDisplay.length === newValue.length + 1) {
    return prevParsed.slice(0, -1);
  }

  return parseAmountInput(newValue);
}