import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  FileDown,
  HandCoins,
  Paperclip,
  Plus,
  Users,
} from "lucide-react";

import AddExpenseForm from "@/components/add-expense-form";
import { Card, CardHeader } from "@/components/card";
import { CategoryIcon } from "@/components/category-icon";
import DeleteExpenseButton from "@/components/delete-expense-button";
import DeleteGroupButton from "@/components/delete-group-button";
import { GroupLogo } from "@/components/group-logo";
import InviteLinkCard from "@/components/invite-link-card";
import SettlePanel from "@/components/settle-panel";
import Toast from "@/components/toast";
import { getCurrentUser } from "@/lib/auth";
import { ApiError, apiFetch } from "@/lib/server-api";
import {
  formatCurrency,
  formatDate,
  formatDateTime,
  formatSignedCurrency,
} from "@/lib/format";
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
      apiFetch<{ settlements: Settlement[] }>(
        `/api/v1/groups/${id}/settlements`,
      ),
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

  const myBalance =
    balances.find((b) => b.userId === user.id)?.balance ?? "0.00";
  const pendingSettlements = suggestions.filter(
    (s) => s.fromUserId === user.id,
  );

  const debtsToMe = suggestions
    .filter((s) => s.toUserId === user.id)
    .map((s) => {
      const name = members.find((m) => m.id === s.fromUserId)?.name;
      return name
        ? tr(dict.group.receiveFromItem, {
            name,
            amount: formatCurrency(s.amount, group.currency),
          })
        : null;
    })
    .filter((d): d is string => d !== null);
  const debtsText =
    debtsToMe.length > 0
      ? debtsToMe.length === 1
        ? debtsToMe[0]
        : `${debtsToMe.slice(0, -1).join(", ")} ${dict.common.and} ${
            debtsToMe[debtsToMe.length - 1]
          }`
      : formatCurrency(myBalance, group.currency);

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
          <GroupLogo
            groupId={group.id}
            hasLogo={group.hasLogo}
            name={group.name}
            className="h-12 w-12 rounded-2xl bg-green-50 text-xl font-bold text-green-700"
            imgClassName="h-12 w-12 rounded-2xl object-cover"
          />
          <div>
            <h1 className="text-display font-bold leading-tight text-slate-900">
              {group.name}
            </h1>
            <p className="text-sm text-slate-500">
              {tr(dict.group.memberCount, { n: group.memberCount })} ·{" "}
              {group.currency}
              {group.description ? ` · ${group.description}` : ""}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <div className="rounded-xl border border-slate-200 bg-white px-4 py-2 shadow-card">
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
          {isAdmin && (
            <DeleteGroupButton
              groupId={group.id}
              groupName={group.name}
              dict={dict}
            />
          )}
          <a
            href={`/api/v1/groups/${group.id}/export`}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-sm font-semibold text-slate-700 shadow-sm transition hover:bg-slate-50 active:scale-[0.98]"
          >
            <FileDown className="h-4 w-4" aria-hidden="true" />
            {dict.group.exportReport}
          </a>
          <a
            href="#add-expense"
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-green-700 active:scale-[0.98]"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {dict.group.addExpense}
          </a>
        </div>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div className="flex flex-col gap-6 xl:col-span-2">
          <Card aria-labelledby="balances-heading">
            <div className="flex items-center justify-between">
              <CardHeader id="balances-heading">
                {dict.group.balances}
              </CardHeader>
              <span className="text-xs font-medium text-slate-400">
                {dict.group.updatedFrom}
              </span>
            </div>

            {balances.length === 0 ? (
              <p className="mt-4 text-sm text-slate-500">
                {dict.group.noMembers}
              </p>
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
                          <span className="ml-1.5 text-xs font-normal text-slate-400">
                            {dict.common.you}
                          </span>
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
          </Card>

          <Card aria-labelledby="expenses-heading">
            <CardHeader id="expenses-heading">
              {dict.group.expenses}
            </CardHeader>

            {expenses.length === 0 ? (
              <div className="py-8 text-center">
                <CategoryIcon
                  category="Other"
                  className="mx-auto h-8 w-8 text-slate-300"
                />
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
                        <CategoryIcon
                          category={e.category}
                          className="h-4 w-4"
                        />
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
                      {e.hasReceipt && (
                        <a
                          href={`/api/v1/expenses/${e.id}/receipt`}
                          target="_blank"
                          rel="noreferrer"
                          aria-label={tr(dict.group.viewReceipt, {
                            description: e.description,
                          })}
                          title={dict.group.viewReceiptTitle}
                          className="rounded-lg p-1.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-600"
                        >
                          <Paperclip className="h-4 w-4" aria-hidden="true" />
                        </a>
                      )}
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
          </Card>
        </div>

        <div className="flex flex-col gap-6">
          <Card
            id="add-expense"
            aria-labelledby="add-expense-heading"
          >
            <CardHeader id="add-expense-heading">
              {dict.group.addExpense}
            </CardHeader>
            <div className="mt-4">
              <AddExpenseForm
                groupId={group.id}
                currency={group.currency}
                members={members}
                user={user}
                dict={dict}
              />
            </div>
          </Card>

          <Card aria-labelledby="settle-heading">
            <CardHeader
              id="settle-heading"
              className="flex items-center gap-2"
            >
              <HandCoins
                className="h-5 w-5 text-green-700"
                aria-hidden="true"
              />
              {dict.group.settleUp}
            </CardHeader>
            <div className="mt-3">
              {suggestions.length === 0 ? (
                <p className="text-sm text-slate-500">
                  {dict.group.nothingToSettle}
                </p>
              ) : pendingSettlements.length === 0 && !isAdmin && myBalance !== "0.00" ? (
                <p className="text-sm text-slate-500">
                  {tr(dict.group.youHaveToReceive, { debts: debtsText })}
                </p>
              ) : (
                <SettlePanel
                  groupId={group.id}
                  members={members}
                  myUserId={user.id}
                  suggestions={suggestions}
                  isAdmin={isAdmin}
                  dict={dict}
                />
              )}
            </div>
          </Card>

          <Card aria-labelledby="members-heading">
            <CardHeader
              id="members-heading"
              className="flex items-center gap-2"
            >
              <Users className="h-5 w-5 text-slate-400" aria-hidden="true" />
              {dict.group.members}
            </CardHeader>

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
                          <span className="ml-1.5 text-xs font-normal text-slate-400">
                            {dict.common.you}
                          </span>
                        )}
                      </p>
                      <p className="truncate text-xs text-slate-500">
                        {m.email}
                      </p>
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
                <p className="text-sm font-medium text-slate-700">
                  {dict.group.inviteMember}
                </p>
                <div className="mt-2">
                  <InviteLinkCard groupId={group.id} dict={dict} />
                </div>
              </div>
            )}
          </Card>

          {settlements.length > 0 && (
            <Card aria-labelledby="settlement-history-heading">
              <CardHeader id="settlement-history-heading">
                {dict.group.settlementHistory}
              </CardHeader>
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
                      <span className="text-slate-400">
                        {" "}
                        · {formatDateTime(s.settledAt)}
                      </span>
                    </p>
                    <span className="shrink-0 text-sm font-semibold text-slate-900">
                      {formatCurrency(s.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </div>
      </div>

      <Toast key={success} success={success} dict={dict} />
    </div>
  );
}
