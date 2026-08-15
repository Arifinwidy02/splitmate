import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  HandCoins,
  Plus,
  Receipt,
  Users,
} from "lucide-react";

import AddExpenseForm from "@/components/add-expense-form";
import DeleteExpenseButton from "@/components/delete-expense-button";
import DeleteGroupButton from "@/components/delete-group-button";
import InviteForm from "@/components/invite-form";
import SettlePanel from "@/components/settle-panel";
import Toast from "@/components/toast";
import { getCurrentUser } from "@/lib/auth";
import { ApiError, apiFetch } from "@/lib/server-api";
import { formatCurrency, formatDate, formatDateTime, formatSignedCurrency } from "@/lib/format";
import type {
  BalanceMember,
  ExpenseListItem,
  GroupSummary,
  Member,
  Settlement,
  Suggestion,
} from "@/lib/api";
import { getDict } from "@/lib/i18n";
import { tr } from "@/lib/i18n/tr";

export const dynamic = "force-dynamic";

export default async function GroupDetailPage({
  params,
  searchParams,
}: PageProps<"/groups/[id]">) {
  const { id } = await params;
  const sp = await searchParams;
  const success = typeof sp.success === "string" ? sp.success : undefined;
  const dict = await getDict();

  const user = await getCurrentUser();

  let group: GroupSummary;
  let members: Member[];
  let expenses: ExpenseListItem[];
  let balances: BalanceMember[];
  let suggestions: Suggestion[];
  let settlements: Settlement[];

  try {
    const [g, m, e, b, s, st] = await Promise.all([
      apiFetch<{ group: GroupSummary }>(`/api/v1/groups/${id}`),
      apiFetch<{ members: Member[] }>(`/api/v1/groups/${id}/members`),
      apiFetch<{ expenses: ExpenseListItem[] }>(
        `/api/v1/groups/${id}/expenses?limit=50`,
      ),
      apiFetch<{ members: BalanceMember[] }>(`/api/v1/groups/${id}/balances`),
      apiFetch<{ settlements: Suggestion[] }>(
        `/api/v1/groups/${id}/settlement-suggestions`,
      ),
      apiFetch<{ settlements: Settlement[] }>(`/api/v1/groups/${id}/settlements`),
    ]);
    group = g.group;
    members = m.members;
    expenses = e.expenses;
    balances = b.members;
    suggestions = s.settlements;
    settlements = st.settlements;
  } catch (err) {
    if (err instanceof ApiError && err.code === "GROUP_NOT_FOUND") {
      notFound();
    }
    throw err;
  }

  const myBalance = balances.find((b) => b.userId === user.id)?.balance ?? "0.00";
  const pendingSettlements = suggestions.filter((s) => s.fromUserId === user.id);
  const isAdmin = group.role === "admin";

  return (
    <div className="mx-auto w-full max-w-[1440px]">
      <Link
        href="/groups"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-slate-500 transition hover:text-slate-900"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        {dict.group.allGroups}
      </Link>

      <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-green-50 text-xl font-bold text-green-700">
            {group.name.charAt(0).toUpperCase()}
          </span>
          <div>
            <h1 className="text-[32px] font-bold leading-tight text-slate-900">
              {group.name}
            </h1>
            <p className="text-sm text-slate-500">
              {tr(dict.group.memberCount, { n: group.memberCount })} · {group.currency}
              {group.description ? ` · ${group.description}` : ""}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <div className="rounded-xl border border-slate-200 bg-white px-4 py-2 shadow-sm">
            <p className="text-xs text-slate-500">{dict.group.yourBalance}</p>
            <p
              className={`text-lg font-bold ${
                Number(myBalance) > 0
                  ? "text-green-700"
                  : Number(myBalance) < 0
                    ? "text-red-600"
                    : "text-slate-900"
              }`}
            >
              {formatSignedCurrency(myBalance, group.currency)}
            </p>
          </div>
          {isAdmin && <DeleteGroupButton groupId={group.id} groupName={group.name} dict={dict} />}
          <a
            href="#add-expense"
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-green-700"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {dict.group.addExpense}
          </a>
        </div>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div className="flex flex-col gap-6 xl:col-span-2">
          <section
            aria-labelledby="balances-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <div className="flex items-center justify-between">
              <h2 id="balances-heading" className="text-xl font-semibold text-slate-900">
                {dict.group.balances}
              </h2>
              <span className="text-xs font-medium text-slate-400">
                {dict.group.updatedFrom}
              </span>
            </div>

            {balances.length === 0 ? (
              <p className="mt-4 text-sm text-slate-500">{dict.group.noMembers}</p>
            ) : (
              <ul className="mt-4 flex flex-col">
                {balances.map((b, i) => (
                  <li
                    key={b.userId}
                    className={`flex items-center justify-between gap-3 py-3 ${
                      i > 0 ? "border-t border-slate-100" : ""
                    }`}
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-slate-100 text-sm font-semibold text-slate-600">
                        {b.name.charAt(0).toUpperCase()}
                      </span>
                      <p className="truncate text-sm font-medium text-slate-900">
                        {b.name}
                        {b.userId === user.id && (
                          <span className="ml-1.5 text-xs font-normal text-slate-400">{dict.common.you}</span>
                        )}
                      </p>
                    </div>
                    <span
                      className={`shrink-0 text-sm font-semibold ${
                        Number(b.balance) > 0
                          ? "text-green-700"
                          : Number(b.balance) < 0
                            ? "text-red-600"
                            : "text-slate-500"
                      }`}
                    >
                      {Number(b.balance) === 0
                        ? dict.group.settled
                        : formatSignedCurrency(b.balance, group.currency)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section
            aria-labelledby="expenses-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="expenses-heading" className="text-xl font-semibold text-slate-900">
              {dict.group.expenses}
            </h2>

            {expenses.length === 0 ? (
              <div className="py-8 text-center">
                <Receipt className="mx-auto h-8 w-8 text-slate-300" aria-hidden="true" />
                <p className="mt-2 text-sm text-slate-500">
                  {dict.group.noExpenses}
                </p>
              </div>
            ) : (
              <ul className="mt-4 flex flex-col">
                {expenses.map((e, i) => (
                  <li
                    key={e.id}
                    className={`flex items-center justify-between gap-3 py-3 ${
                      i > 0 ? "border-t border-slate-100" : ""
                    }`}
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
                        <Receipt className="h-4 w-4" aria-hidden="true" />
                      </span>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-slate-900">
                          {e.description}
                        </p>
                        <p className="truncate text-xs text-slate-500">
                          {tr(dict.group.expenseMeta, {
                            payer: e.payerName,
                            category: e.category,
                            date: formatDate(e.expenseDate),
                          })}
                        </p>
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      <span className="text-sm font-semibold text-slate-900">
                        {formatCurrency(e.amount, e.currency)}
                      </span>
                      {e.createdBy === user.id && (
                        <DeleteExpenseButton
                          groupId={group.id}
                          expenseId={e.id}
                          description={e.description}
                          dict={dict}
                        />
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>

        <div className="flex flex-col gap-6">
          <section
            id="add-expense"
            aria-labelledby="add-expense-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="add-expense-heading" className="text-xl font-semibold text-slate-900">
              {dict.group.addExpense}
            </h2>
            <div className="mt-4">
              <AddExpenseForm
                groupId={group.id}
                currency={group.currency}
                members={members}
                user={user}
                dict={dict}
              />
            </div>
          </section>

          <section
            aria-labelledby="settle-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="settle-heading" className="flex items-center gap-2 text-xl font-semibold text-slate-900">
              <HandCoins className="h-5 w-5 text-green-700" aria-hidden="true" />
              {dict.group.settleUp}
            </h2>
            <div className="mt-3">
              {suggestions.length === 0 ? (
                <p className="text-sm text-slate-500">
                  {dict.group.nothingToSettle}
                </p>
              ) : pendingSettlements.length === 0 && myBalance !== "0.00" ? (
                <p className="text-sm text-slate-500">
                  {tr(dict.group.youHaveToReceive, {
                    amount: formatCurrency(myBalance, group.currency),
                    names: members
                      .filter((m) => m.id !== user.id)
                      .map((m) => m.name)
                      .join(", "),
                  })}
                </p>
              ) : (
                <SettlePanel
                  groupId={group.id}
                  members={members}
                  myUserId={user.id}
                  suggestions={suggestions}
                  dict={dict}
                />
              )}
            </div>
          </section>

          <section
            aria-labelledby="members-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="members-heading" className="flex items-center gap-2 text-xl font-semibold text-slate-900">
              <Users className="h-5 w-5 text-slate-400" aria-hidden="true" />
              {dict.group.members}
            </h2>

            <ul className="mt-4 flex flex-col">
              {members.map((m, i) => (
                <li
                  key={m.id}
                  className={`flex items-center justify-between gap-3 py-2.5 ${
                    i > 0 ? "border-t border-slate-100" : ""
                  }`}
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-100 text-sm font-semibold text-slate-600">
                      {m.name.charAt(0).toUpperCase()}
                    </span>
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-slate-900">
                        {m.name}
                        {m.id === user.id && (
                          <span className="ml-1.5 text-xs font-normal text-slate-400">{dict.common.you}</span>
                        )}
                      </p>
                      <p className="truncate text-xs text-slate-500">{m.email}</p>
                    </div>
                  </div>
                  {m.role === "admin" && (
                    <span className="shrink-0 rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-700">
                      {dict.group.admin}
                    </span>
                  )}
                </li>
              ))}
            </ul>

            {isAdmin && (
              <div className="mt-4 border-t border-slate-100 pt-4">
                <p className="text-sm font-medium text-slate-700">{dict.group.inviteMember}</p>
                <div className="mt-2">
                  <InviteForm groupId={group.id} dict={dict} />
                </div>
              </div>
            )}
          </section>

          {settlements.length > 0 && (
            <section
              aria-labelledby="settlement-history-heading"
              className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
            >
              <h2
                id="settlement-history-heading"
                className="text-xl font-semibold text-slate-900"
              >
                {dict.group.settlementHistory}
              </h2>
              <ul className="mt-3 flex flex-col">
                {settlements.map((s, i) => (
                  <li
                    key={s.id}
                    className={`flex items-center justify-between gap-3 py-2.5 ${
                      i > 0 ? "border-t border-slate-100" : ""
                    }`}
                  >
                    <p className="min-w-0 truncate text-sm text-slate-700">
                      <span className="font-medium">{s.payerName}</span>{" "}
                      {dict.group.paid}{" "}
                      <span className="font-medium">{s.receiverName}</span>
                      <span className="text-slate-400"> · {formatDateTime(s.settledAt)}</span>
                    </p>
                    <span className="shrink-0 text-sm font-semibold text-slate-900">
                      {formatCurrency(s.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </div>
      </div>

      <Toast key={success} success={success} dict={dict} />
    </div>
  );
}
