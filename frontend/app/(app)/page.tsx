import Link from "next/link";
import {
  ArrowDownCircle,
  ArrowUpCircle,
  ArrowRight,
  Plus,
  Receipt,
  Scale,
  Wallet,
} from "lucide-react";

import { CategoryIcon } from "@/components/category-icon";
import { GroupLogo } from "@/components/group-logo";
import Toast from "@/components/toast";
import { getCurrentUser } from "@/lib/auth";
import { apiFetch } from "@/lib/server-api";
import { formatCurrency, formatDate, formatSignedCurrency } from "@/lib/format";
import type { DashboardData } from "@/lib/api";
import type { Dict } from "@/lib/i18n/id";
import { getDict } from "@/lib/i18n";
import { tr } from "@/lib/i18n/tr";

function SummaryCard({
  label,
  amount,
  currency,
  positive,
  negative,
  icon: Icon,
}: {
  label: string;
  amount: string;
  currency: string;
  positive?: boolean;
  negative?: boolean;
  icon: typeof Scale;
}) {
  const tone = positive
    ? "text-green-700"
    : negative
      ? "text-red-600"
      : "text-slate-900";

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-2 text-sm font-medium text-slate-500">
        <Icon className="h-4 w-4" aria-hidden="true" />
        {label}
      </div>
      <p className={`mt-2 text-[28px] font-bold leading-tight ${tone}`}>
        {formatCurrency(amount, currency)}
      </p>
    </div>
  );
}

function CategoryChart({
  categories,
  currency,
  dict,
}: {
  categories: DashboardData["categories"];
  currency: string;
  dict: Dict;
}) {
  if (categories.length === 0) {
    return (
      <p className="text-sm text-slate-500">{dict.dashboard.noCategories}</p>
    );
  }

  const max = Math.max(...categories.map((c) => Number(c.total)));

  return (
    <ul className="flex flex-col gap-4">
      {categories.map(({ category, total }) => {
        const pct = max > 0 ? Math.round((Number(total) / max) * 100) : 0;
        return (
          <li key={category}>
            <div className="flex items-center justify-between text-sm">
              <span className="flex items-center gap-2 font-medium text-slate-700">
                <CategoryIcon category={category} className="h-4 w-4 text-green-700" />
                {category}
              </span>
              <span className="font-semibold text-slate-900">{formatCurrency(total, currency)}</span>
            </div>
            <div
              className="mt-1.5 h-2 rounded-full bg-slate-100"
              role="img"
              aria-label={tr(dict.dashboard.categoryAria, { category, pct })}
            >
              <div
                className="h-2 rounded-full bg-green-600"
                style={{ width: `${pct}%` }}
              />
            </div>
          </li>
        );
      })}
    </ul>
  );
}

export default async function DashboardPage({
  searchParams,
}: {
  searchParams: Promise<{ success?: string }>;
}) {
  const { success } = await searchParams;
  const user = await getCurrentUser();
  const dict = await getDict();
  const { summary, groups, recentExpenses, categories } = await apiFetch<DashboardData>(
    "/api/v1/dashboard",
  );

  const hasData = groups.length > 0 || recentExpenses.length > 0;

  return (
    <div className="mx-auto w-full max-w-[1440px]">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-[32px] font-bold text-slate-900">
            {tr(dict.dashboard.welcome, { name: user.name.split(" ")[0] })}
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            {dict.dashboard.subtitle}
          </p>
        </div>
        <Link
          href="/groups"
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-green-700"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          {dict.dashboard.addExpense}
        </Link>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          label={dict.dashboard.youAreOwed}
          amount={summary.owedToUser}
          currency="IDR"
          positive
          icon={ArrowDownCircle}
        />
        <SummaryCard
          label={dict.dashboard.youOwe}
          amount={summary.userOwes}
          currency="IDR"
          negative
          icon={ArrowUpCircle}
        />
        <SummaryCard label={dict.dashboard.netBalance} amount={summary.netBalance} currency="IDR" icon={Scale} />
        <SummaryCard label={dict.dashboard.totalExpense} amount={summary.totalExpense} currency="IDR" icon={Wallet} />
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-2">
        <section
          aria-labelledby="recent-groups-heading"
          className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
        >
          <div className="flex items-center justify-between">
            <h2 id="recent-groups-heading" className="text-xl font-semibold text-slate-900">
              {dict.dashboard.recentGroups}
            </h2>
            <Link href="/groups" className="inline-flex items-center gap-1 text-sm font-medium text-green-700 hover:underline">
              {dict.dashboard.viewAll} <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
            </Link>
          </div>

          <ul className="mt-4 flex flex-col">
            {groups.length === 0 ? (
              <EmptyHint
                title={dict.dashboard.noGroupsTitle}
                body={dict.dashboard.noGroupsBody}
              />
            ) : (
              groups.slice(0, 4).map((g, i) => (
                <li
                  key={g.id}
                  className={
                    i > 0 ? "border-t border-slate-100" : undefined
                  }
                >
                  <Link
                    href={`/groups/${g.id}`}
                    className="flex items-center justify-between gap-3 px-1 py-3 transition hover:bg-slate-50 rounded-lg"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <GroupLogo groupId={g.id} hasLogo={g.hasLogo} name={g.name} />
                      <div className="min-w-0">
                        <p className="truncate font-medium text-slate-900">{g.name}</p>
                        <p className="text-sm text-slate-500">{tr(dict.dashboard.memberCount, { n: g.memberCount })}</p>
                      </div>
                    </div>
                    <span
                      className={`text-sm font-semibold ${
                        Number(g.balance) > 0
                          ? "text-green-700"
                          : Number(g.balance) < 0
                            ? "text-red-600"
                            : "text-slate-500"
                      }`}
                    >
                      {formatSignedCurrency(g.balance, g.currency)}
                    </span>
                  </Link>
                </li>
              ))
            )}
          </ul>
        </section>

        <section
          aria-labelledby="recent-expenses-heading"
          className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
        >
          <div className="flex items-center justify-between">
            <h2 id="recent-expenses-heading" className="text-xl font-semibold text-slate-900">
              {dict.dashboard.recentExpenses}
            </h2>
          </div>

          <ul className="mt-4 flex flex-col">
            {recentExpenses.length === 0 ? (
              <EmptyHint
                title={dict.dashboard.noExpensesTitle}
                body={dict.dashboard.noExpensesBody}
              />
            ) : (
              recentExpenses.map((e, i) => (
                <li key={e.id} className={i > 0 ? "border-t border-slate-100" : undefined}>
                  <Link
                    href={`/groups/${e.groupId}`}
                    className="flex items-center justify-between gap-3 px-1 py-3 transition hover:bg-slate-50 rounded-lg"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
                        <Receipt className="h-4.5 w-4.5" aria-hidden="true" />
                      </span>
                      <div className="min-w-0">
                        <p className="truncate font-medium text-slate-900">{e.description}</p>
                        <p className="truncate text-sm text-slate-500">
                          {e.groupName} · {e.payerName} · {formatDate(e.expenseDate)}
                        </p>
                      </div>
                    </div>
                    <span className="shrink-0 text-sm font-semibold text-slate-900">
                      {formatCurrency(e.amount)}
                    </span>
                  </Link>
                </li>
              ))
            )}
          </ul>
        </section>
      </div>

      {hasData && (
        <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-2">
          <section
            aria-labelledby="balance-overview-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="balance-overview-heading" className="text-xl font-semibold text-slate-900">
              {dict.dashboard.balanceOverview}
            </h2>

            <dl className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <BalanceRow label={dict.dashboard.owedToYou} amount={summary.owedToUser} positive />
              <BalanceRow label={dict.dashboard.youOwe} amount={summary.userOwes} negative />
              <BalanceRow label={dict.dashboard.settled} amount={summary.settledAmount} neutral />
              <BalanceRow label={dict.dashboard.netBalance} amount={summary.netBalance} />
            </dl>
          </section>

          <section
            aria-labelledby="categories-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="categories-heading" className="text-xl font-semibold text-slate-900">
              {dict.dashboard.expenseCategories}
            </h2>
            <div className="mt-5">
              <CategoryChart categories={categories} currency="IDR" dict={dict} />
            </div>
          </section>
        </div>
      )}
      <Toast success={success} dict={dict} />
    </div>
  );
}

function BalanceRow({
  label,
  amount,
  positive,
  negative,
  neutral,
}: {
  label: string;
  amount: string;
  positive?: boolean;
  negative?: boolean;
  neutral?: boolean;
}) {
  const tone = neutral
    ? "text-slate-700"
    : positive
      ? "text-green-700"
      : negative
        ? "text-red-600"
        : Number(amount) > 0
          ? "text-green-700"
          : Number(amount) < 0
            ? "text-red-600"
            : "text-slate-900";

  return (
    <div className="rounded-xl border border-slate-100 bg-slate-50 px-4 py-3">
      <dt className="text-sm text-slate-500">{label}</dt>
      <dd className={`mt-0.5 text-lg font-bold ${tone}`}>{formatCurrency(amount)}</dd>
    </div>
  );
}

function EmptyHint({ title, body }: { title: string; body: string }) {
  return (
    <li className="py-8 text-center">
      <p className="font-medium text-slate-700">{title}</p>
      <p className="mt-1 text-sm text-slate-500">{body}</p>
    </li>
  );
}
