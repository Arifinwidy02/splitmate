import Image from "next/image";
import Link from "next/link";

import LanguageSwitcher from "@/components/language-switcher";
import type { Locale } from "@/lib/i18n/constants";

export default function AuthHeader({ locale }: { locale: Locale }) {
  return (
    <header className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3 sm:px-6">
      <Link href="/login" className="flex items-center gap-2" aria-label="SplitMate">
        <Image
          src="/splitmate_logo.png"
          alt=""
          width={32}
          height={32}
          className="h-8 w-8 rounded-lg"
        />
        <span className="text-lg font-bold text-slate-900">SplitMate</span>
      </Link>
      <LanguageSwitcher current={locale} />
    </header>
  );
}