export type Locale = "id" | "en";

export const LOCALE_COOKIE = "lang";
export const DEFAULT_LOCALE: Locale = "id";

export function getClientLocale(): Locale {
  if (typeof document === "undefined") return DEFAULT_LOCALE;
  const match = document.cookie.match(/(?:^|;\s*)lang=([^;]*)/);
  return match?.[1] === "en" ? "en" : DEFAULT_LOCALE;
}