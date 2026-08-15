import { cookies } from "next/headers";

import { DEFAULT_LOCALE, LOCALE_COOKIE, type Locale } from "./constants";
import { dictionaries } from "./dictionaries";
import type { Dict } from "./id";

export type { Locale };
export { LOCALE_COOKIE, DEFAULT_LOCALE };
export { dictionaries };

export async function getLocale(): Promise<Locale> {
  const store = await cookies();
  const value = store.get(LOCALE_COOKIE)?.value;
  return value === "en" ? "en" : DEFAULT_LOCALE;
}

export async function getDict(): Promise<Dict> {
  return dictionaries[await getLocale()];
}