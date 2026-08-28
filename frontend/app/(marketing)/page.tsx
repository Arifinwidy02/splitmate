import type { Metadata } from "next";
import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  ArrowDownCircle,
  ArrowRight,
  ArrowUpCircle,
  Check,
  FileDown,
  Paperclip,
  Scale,
  Wallet,
} from "lucide-react";

import { Card, CardHeader } from "@/components/card";
import { CategoryIcon } from "@/components/category-icon";
import { HeroCarousel } from "@/components/hero-carousel";
import Reveal from "@/components/reveal";
import { formatCurrency, formatSignedCurrency } from "@/lib/format";
import { getDict } from "@/lib/i18n";
import { ACCESS_TOKEN_COOKIE } from "@/lib/tokens";

export const metadata: Metadata = {
  title: "SplitMate: track shared expenses and settle up easily",
  description:
    "SplitMate keeps shared expenses clear: record who paid, see who owes whom, and settle up with friends, family, and teams.",
};

const previewBalances = [
  { name: "Sinta", amount: "625000" },
  { name: "Dimas", amount: "-860000" },
  { name: "Maya", amount: "235000" },
  { name: "Raka", amount: "0" },
] as const;

const previewExpenses = [
  { description: "Makan malam di Warung", payer: "Sinta", category: "Food & Drinks", amount: "425000" },
  { description: "Bensin dan parkir", payer: "Dimas", category: "Transportation", amount: "150000" },
  { description: "Tiket wisata", payer: "Maya", category: "Entertainment", amount: "380000" },
] as const;

const simplifiedDebts = [
  { from: "B", to: "A", amount: "400000" },
  { from: "C", to: "A", amount: "300000" },
] as const;

export default async function LandingPage() {
  const cookieStore = await cookies();
  if (cookieStore.get(ACCESS_TOKEN_COOKIE)) {
    redirect("/dashboard");
  }

  const dict = await getDict();
  const d = dict.dashboard;
  const summaryCards = [
    { label: d.youAreOwed, amount: "1485000", tone: "text-green-700", icon: ArrowDownCircle },
    { label: d.youOwe, amount: "325000", tone: "text-red-600", icon: ArrowUpCircle },
    { label: d.netBalance, amount: "1160000", tone: "text-slate-900", icon: Scale },
    { label: d.totalExpense, amount: "4820000", tone: "text-slate-900", icon: Wallet },
  ] as const;

  const steps = [
    { title: dict.landing.step1Title, body: dict.landing.step1Body },
    { title: dict.landing.step2Title, body: dict.landing.step2Body },
    { title: dict.landing.step3Title, body: dict.landing.step3Body },
  ] as const;

  const heroWords = dict.landing.heroTitle.split(" ");

  return (
    <div className="mx-auto w-full max-w-[1440px] px-4 pb-16 sm:px-6 sm:pb-24 lg:px-8">
      <section
        aria-label={dict.landing.heroAria}
        className="relative isolate flex min-h-[560px] items-center overflow-hidden rounded-3xl sm:min-h-[640px]"
      >
        <HeroCarousel />

        <div
          aria-hidden="true"
          className="absolute inset-0 bg-gradient-to-b from-white/95 via-white/85 to-white/60 sm:bg-gradient-to-r sm:from-white sm:via-white/90 sm:to-transparent"
        />

        <div className="relative w-full py-16 pl-6 pr-6 sm:py-24 sm:pl-12 lg:pl-16">
          <div className="max-w-xl">
            <h1 className="text-5xl font-bold leading-[1.05] tracking-tighter text-slate-900 sm:text-6xl">
              {heroWords.map((word, i) => (
                <span
                  key={`${word}-${i}`}
                  className={`animate-hero-word inline-block ${
                    i === heroWords.length - 1 ? "text-green-600" : ""
                  }`}
                  style={{ animationDelay: `${i * 120}ms` }}
                >
                  {word}{" "}
                </span>
              ))}
            </h1>
            <p
              className="animate-hero-word mt-6 max-w-md text-lg leading-relaxed text-slate-700"
              style={{ animationDelay: "420ms" }}
            >
              {dict.landing.heroSubtitle}
            </p>
            <div
              className="animate-hero-word mt-8 flex flex-col gap-3 sm:flex-row"
              style={{ animationDelay: "540ms" }}
            >
              <Link
                href="/register"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-5 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-green-700 active:scale-[0.98]"
              >
                {dict.auth.createAccount}
              </Link>
              <Link
                href="/login"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white/90 px-5 py-3 text-sm font-semibold text-slate-700 shadow-sm backdrop-blur-sm transition hover:bg-white active:scale-[0.98]"
              >
                {dict.auth.signIn}
                <ArrowRight className="h-4 w-4" aria-hidden="true" />
              </Link>
            </div>
          </div>
        </div>

        <div
          aria-hidden="true"
          className="chip-float absolute bottom-14 right-6 hidden lg:block"
        >
          <div className="chip-in flex items-center gap-2.5 rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 shadow-lg">
            <span className="flex h-7 w-7 items-center justify-center rounded-full bg-green-50">
              <Check className="h-4 w-4 text-green-700" />
            </span>
            <span className="text-sm font-medium text-slate-800">
              {dict.landing.heroChip}
            </span>
          </div>
        </div>
      </section>

      <Reveal className="relative z-10 -mt-10 sm:-mt-16">
        <Card
          aria-labelledby="landing-preview-heading"
          className="shadow-[0_24px_70px_-24px_rgba(22,163,74,0.25)]"
        >
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <CardHeader id="landing-preview-heading">{dict.landing.previewTitle}</CardHeader>
            <p className="text-xs text-slate-500">{dict.landing.previewCaption}</p>
          </div>

          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {summaryCards.map((item) => (
              <div
                key={item.label}
                className="rounded-xl border border-slate-100 bg-slate-50 p-4"
              >
                <div className="flex items-center gap-2 text-sm font-medium text-slate-500">
                  <item.icon className="h-4 w-4" aria-hidden="true" />
                  {item.label}
                </div>
                <p className={`mt-2 text-2xl font-bold ${item.tone}`}>
                  {formatCurrency(item.amount)}
                </p>
              </div>
            ))}
          </div>

          <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-2">
            <div>
              <h3 className="text-base font-semibold text-slate-900">
                {dict.landing.previewBalancesTitle}
                <span className="ml-2 text-sm font-normal text-slate-500">
                  {dict.landing.previewMembers}
                </span>
              </h3>
              <ul className="mt-3 flex flex-col">
                {previewBalances.map((b, i) => (
                  <li
                    key={b.name}
                    className={`flex items-center justify-between gap-3 py-2.5 ${
                      i > 0 ? "border-t border-slate-100" : ""
                    }`}
                  >
                    <span className="flex min-w-0 items-center gap-3">
                      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-100 text-sm font-semibold text-slate-600">
                        {b.name.charAt(0)}
                      </span>
                      <span className="text-sm font-medium text-slate-900">{b.name}</span>
                    </span>
                    <span
                      className={`shrink-0 text-sm font-semibold ${
                        Number(b.amount) > 0
                          ? "text-green-700"
                          : Number(b.amount) < 0
                            ? "text-red-600"
                            : "text-slate-500"
                      }`}
                    >
                      {Number(b.amount) === 0
                        ? d.settled
                        : formatSignedCurrency(b.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>

            <div>
              <h3 className="text-base font-semibold text-slate-900">
                {dict.landing.previewExpensesTitle}
              </h3>
              <ul className="mt-3 flex flex-col">
                {previewExpenses.map((e, i) => (
                  <li
                    key={e.description}
                    className={`flex items-center justify-between gap-3 py-2.5 ${
                      i > 0 ? "border-t border-slate-100" : ""
                    }`}
                  >
                    <span className="flex min-w-0 items-center gap-3">
                      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
                        <CategoryIcon category={e.category} className="h-4 w-4" />
                      </span>
                      <span className="min-w-0">
                        <span className="block truncate text-sm font-medium text-slate-900">
                          {e.description}
                        </span>
                        <span className="block truncate text-xs text-slate-500">
                          {e.payer} · {e.category}
                        </span>
                      </span>
                    </span>
                    <span className="shrink-0 text-sm font-semibold text-slate-900">
                      {formatCurrency(e.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </Card>
      </Reveal>

      <Reveal>
        <section
          aria-labelledby="landing-how-heading"
          className="mt-16 border-t border-slate-200 pt-16 sm:mt-24 sm:pt-24"
        >
          <h2
            id="landing-how-heading"
            className="max-w-2xl text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl"
          >
            {dict.landing.howTitle}
          </h2>

          <div className="relative mt-12">
            <div
              aria-hidden="true"
              className="absolute left-[16%] right-[16%] top-5 hidden h-px bg-slate-200 lg:block"
            />
            <ol className="grid gap-10 lg:grid-cols-3 lg:gap-8">
              {steps.map((step, i) => (
                <li key={step.title} className="relative">
                  <span className="relative z-10 flex h-10 w-10 items-center justify-center rounded-full bg-green-600 text-sm font-bold text-white shadow-sm">
                    {i + 1}
                  </span>
                  <h3 className="mt-4 text-lg font-semibold text-slate-900">{step.title}</h3>
                  <p className="mt-1.5 leading-relaxed text-slate-600">{step.body}</p>
                </li>
              ))}
            </ol>
          </div>
        </section>
      </Reveal>

      <Reveal>
        <section aria-labelledby="landing-features-heading" className="mt-16 sm:mt-24">
          <h2
            id="landing-features-heading"
            className="max-w-2xl text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl"
          >
            {dict.landing.featuresTitle}
          </h2>

          <div className="mt-10 grid gap-4 sm:grid-cols-2">
            <div className="rounded-2xl border border-slate-200 bg-white p-6">
              <h3 className="text-lg font-semibold text-slate-900">
                {dict.landing.feat1Title}
              </h3>
              <p className="mt-1 text-sm leading-relaxed text-slate-600">
                {dict.landing.feat1Body}
              </p>
              <ul className="mt-5 rounded-xl border border-slate-100 bg-slate-50 px-4 py-2">
                {previewBalances.slice(0, 3).map((b) => (
                  <li
                    key={b.name}
                    className="flex items-center justify-between gap-3 border-t border-slate-200/60 py-2 first:border-t-0"
                  >
                    <span className="text-sm font-medium text-slate-700">{b.name}</span>
                    <span
                      className={`text-sm font-semibold ${
                        Number(b.amount) > 0
                          ? "text-green-700"
                          : Number(b.amount) < 0
                            ? "text-red-600"
                            : "text-slate-500"
                      }`}
                    >
                      {formatSignedCurrency(b.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-white p-6">
              <h3 className="text-lg font-semibold text-slate-900">
                {dict.landing.feat2Title}
              </h3>
              <p className="mt-1 text-sm leading-relaxed text-slate-600">
                {dict.landing.feat2Body}
              </p>
              <ul className="mt-5 flex flex-col gap-2 rounded-xl border border-slate-100 bg-slate-50 px-4 py-3">
                {simplifiedDebts.map((debt) => (
                  <li key={debt.from} className="flex items-center gap-2.5">
                    <span className="flex h-7 w-7 items-center justify-center rounded-full bg-white text-xs font-semibold text-slate-600 shadow-sm">
                      {debt.from}
                    </span>
                    <ArrowRight className="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />
                    <span className="flex h-7 w-7 items-center justify-center rounded-full bg-green-600 text-xs font-semibold text-white shadow-sm">
                      {debt.to}
                    </span>
                    <span className="ml-auto text-sm font-semibold text-slate-900">
                      {formatCurrency(debt.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>

            <div className="rounded-2xl border border-green-100 bg-green-50 p-6">
              <h3 className="text-lg font-semibold text-slate-900">
                {dict.landing.feat3Title}
              </h3>
              <p className="mt-1 text-sm leading-relaxed text-slate-600">
                {dict.landing.feat3Body}
              </p>
              <div className="mt-5 flex gap-2">
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-white text-slate-500 shadow-sm">
                  <Paperclip className="h-4 w-4" aria-hidden="true" />
                </span>
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-white text-green-700 shadow-sm">
                  <FileDown className="h-4 w-4" aria-hidden="true" />
                </span>
              </div>
            </div>

            <div className="rounded-2xl border border-slate-200 bg-slate-100 p-6">
              <h3 className="text-lg font-semibold text-slate-900">
                {dict.landing.feat4Title}
              </h3>
              <p className="mt-1 text-sm leading-relaxed text-slate-600">
                {dict.landing.feat4Body}
              </p>
              <div className="mt-5 inline-flex items-center rounded-lg bg-white p-0.5 text-xs shadow-sm">
                <span className="rounded-lg bg-white px-2.5 py-1.5 font-semibold text-slate-900 shadow-sm">
                  ID
                </span>
                <span className="px-2.5 py-1.5 font-semibold text-slate-500">EN</span>
              </div>
            </div>
          </div>
        </section>
      </Reveal>

      <Reveal>
        <section className="mt-16 sm:mt-24">
          <div className="relative overflow-hidden rounded-2xl bg-green-800 px-6 py-14 text-center sm:px-16 sm:py-16">
            <div
              aria-hidden="true"
              className="pointer-events-none absolute inset-0 bg-[radial-gradient(70%_100%_at_50%_0%,rgba(255,255,255,0.12),transparent)]"
            />
            <h2 className="relative text-3xl font-bold tracking-tight text-white sm:text-4xl">
              {dict.landing.ctaTitle}
            </h2>
            <p className="relative mx-auto mt-3 max-w-md text-sm leading-relaxed text-green-100 sm:text-base">
              {dict.landing.ctaBody}
            </p>
            <div className="relative mt-8 flex flex-col items-center gap-3">
              <Link
                href="/register"
                className="inline-flex items-center justify-center rounded-lg bg-white px-5 py-3 text-sm font-semibold text-green-800 shadow-sm transition hover:bg-green-50 active:scale-[0.98]"
              >
                {dict.auth.createAccount}
              </Link>
              <Link
                href="/login"
                className="text-sm font-medium text-green-100 underline-offset-4 transition hover:text-white hover:underline"
              >
                {dict.auth.signIn}
              </Link>
            </div>
          </div>
        </section>
      </Reveal>
    </div>
  );
}
