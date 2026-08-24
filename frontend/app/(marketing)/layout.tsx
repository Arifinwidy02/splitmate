import Image from "next/image";
import Link from "next/link";

import LanguageSwitcher from "@/components/language-switcher";
import { getDict } from "@/lib/i18n";

export default async function MarketingLayout({ children }: LayoutProps<"/">) {
  const dict = await getDict();

  return (
    <div className="flex min-h-full flex-col bg-slate-50">
      <div
        aria-hidden="true"
        className="grain pointer-events-none fixed inset-0 z-50 opacity-[0.03]"
      />

      <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/85 backdrop-blur-sm">
        <div className="mx-auto flex h-16 w-full max-w-[1440px] items-center justify-between px-4 sm:px-6 lg:px-8">
          <Link href="/" className="flex items-center gap-2" aria-label="SplitMate">
            <Image
              src="/splitmate_logo.png"
              alt=""
              width={32}
              height={32}
              className="h-8 w-8 rounded-lg"
            />
            <span className="text-lg font-bold text-slate-900">SplitMate</span>
          </Link>

          <div className="flex items-center gap-3">
            <LanguageSwitcher current={dict.locale} />
            <Link
              href="/login"
              className="rounded-lg px-3 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-100 hover:text-slate-900"
            >
              {dict.auth.signIn}
            </Link>
            <Link
              href="/register"
              className="hidden rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-green-700 active:scale-[0.98] sm:inline-flex"
            >
              {dict.auth.createAccount}
            </Link>
          </div>
        </div>
      </header>

      <main id="main" className="flex-1">{children}</main>

      <footer className="border-t border-slate-200 bg-white">
        <div className="mx-auto flex w-full max-w-[1440px] flex-col gap-4 px-4 py-8 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
          <div className="flex items-center gap-2">
            <Image
              src="/splitmate_logo.png"
              alt=""
              width={24}
              height={24}
              className="h-6 w-6 rounded-md"
            />
            <span className="text-sm font-semibold text-slate-900">SplitMate</span>
            <span className="text-sm text-slate-500">{dict.landing.footerTagline}</span>
          </div>
          <div className="flex items-center gap-4 text-sm font-medium">
            <Link href="/login" className="text-green-700 transition hover:underline">
              {dict.auth.signIn}
            </Link>
            <Link href="/register" className="text-green-700 transition hover:underline">
              {dict.auth.createAccount}
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
