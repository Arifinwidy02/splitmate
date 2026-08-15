import type { Locale } from "./constants";
import { en } from "./en";
import { id, type Dict } from "./id";

export const dictionaries: Record<Locale, Dict> = { id, en };