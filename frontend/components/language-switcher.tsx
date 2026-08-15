"use client";

import { useRouter } from "next/navigation";

import { LOCALE_COOKIE, type Locale } from "@/lib/i18n/constants";

export default function LanguageSwitcher({ current }: { current: Locale }) {
  const router = useRouter();

  const setLocale = (locale: Locale) => {
    document.cookie = `${LOCALE_COOKIE}=${locale}; path=/; max-age=31536000; samesite=lax`;
    router.refresh();
  };

  const buttonClass = (active: boolean) =>
    `rounded-md px-2 py-1 font-semibold transition ${
      active ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700"
    }`;

  return (
    <div
      role="group"
      aria-label="Language"
      className="flex items-center rounded-lg bg-slate-100 p-0.5 text-xs"
    >
      <button
        type="button"
        onClick={() => setLocale("id")}
        aria-pressed={current === "id"}
        className={buttonClass(current === "id")}
      >
        ID
      </button>
      <button
        type="button"
        onClick={() => setLocale("en")}
        aria-pressed={current === "en"}
        className={buttonClass(current === "en")}
      >
        EN
      </button>
    </div>
  );
}