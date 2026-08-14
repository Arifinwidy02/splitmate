"use client";

import { useState } from "react";
import { useActionState } from "react";

import { createExpense } from "@/app/actions/expenses";
import { EXPENSE_CATEGORIES, type Member, type User } from "@/lib/api";
import { toRFC3339 } from "@/lib/format";

function localDateTimeValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(
    d.getHours(),
  )}:${pad(d.getMinutes())}`;
}

export default function AddExpenseForm({
  groupId,
  currency,
  members,
  user,
}: {
  groupId: string;
  currency: string;
  members: Member[];
  user: User;
}) {
  const [state, action, pending] = useActionState(createExpense.bind(null, groupId), undefined);

  const [splitType, setSplitType] = useState<"equal" | "custom">("equal");
  const [selected, setSelected] = useState<Set<string>>(new Set([user.id]));
  const [customTotals, setCustomTotals] = useState<Record<string, string>>({});
  const [expenseDate, setExpenseDate] = useState<string>(() => localDateTimeValue(new Date()));

  const toggle = (userId: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(userId)) {
        next.delete(userId);
      } else {
        next.add(userId);
      }
      return next;
    });
  };

  return (
    <form action={action} className="flex flex-col gap-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <label htmlFor="description" className="text-sm font-medium text-slate-700">
            Description
          </label>
          <input
            id="description"
            name="description"
            type="text"
            required
            maxLength={255}
            placeholder="e.g. Dinner at Warung"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="amount" className="text-sm font-medium text-slate-700">
            Amount
          </label>
          <input
            id="amount"
            name="amount"
            type="text"
            required
            inputMode="decimal"
            placeholder="0.00"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="category" className="text-sm font-medium text-slate-700">
            Category
          </label>
          <select
            id="category"
            name="category"
            defaultValue="Food & Drinks"
            className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          >
            {EXPENSE_CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="paidBy" className="text-sm font-medium text-slate-700">
            Paid by
          </label>
          <select
            id="paidBy"
            name="paidBy"
            defaultValue={user.id}
            className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          >
            {members.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="expenseDate" className="text-sm font-medium text-slate-700">
            Date
          </label>
          <input
            id="expenseDate"
            name="expenseDate"
            type="datetime-local"
            required
            value={expenseDate}
            onChange={(e) => setExpenseDate(e.target.value)}
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>
      </div>

      <input type="hidden" name="expenseDateRfc" value={toRFC3339(expenseDate)} />
      <input type="hidden" name="currency" value={currency} />

      <fieldset>
        <legend className="text-sm font-medium text-slate-700">Split method</legend>
        <div className="mt-2 grid grid-cols-2 gap-2 rounded-lg bg-slate-100 p-1">
          {(
            [
              ["equal", "Split equally"],
              ["custom", "Custom amounts"],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => setSplitType(value)}
              aria-pressed={splitType === value}
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition ${
                splitType === value
                  ? "bg-white text-slate-900 shadow-sm"
                  : "text-slate-500 hover:text-slate-700"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </fieldset>

      <input type="hidden" name="splitType" value={splitType} />

      <fieldset>
        <legend className="text-sm font-medium text-slate-700">Split between</legend>
        <ul className="mt-2 flex flex-col">
          {members.map((m, i) => (
            <li
              key={m.id}
              className={i > 0 ? "border-t border-slate-100" : undefined}
            >
              <label
                htmlFor={`participant-${m.id}`}
                className="flex cursor-pointer items-center justify-between gap-3 py-2.5"
              >
                <span className="flex items-center gap-2 text-sm text-slate-800">
                  <input
                    id={`participant-${m.id}`}
                    type="checkbox"
                    name="participant"
                    value={m.id}
                    checked={selected.has(m.id)}
                    onChange={() => toggle(m.id)}
                    className="h-4 w-4 rounded border-slate-300 accent-green-600"
                  />
                  {m.name}
                  {m.id === user.id && (
                    <span className="text-xs text-slate-400">(you)</span>
                  )}
                </span>
                {splitType === "custom" && selected.has(m.id) && (
                  <input
                    type="text"
                    inputMode="decimal"
                    name={`split-${m.id}`}
                    value={customTotals[m.id] ?? ""}
                    onChange={(e) =>
                      setCustomTotals((prev) => ({ ...prev, [m.id]: e.target.value }))
                    }
                    placeholder="0.00"
                    aria-label={`Share for ${m.name}`}
                    className="w-28 rounded-lg border border-slate-200 px-2 py-1 text-right text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
                  />
                )}
              </label>
            </li>
          ))}
        </ul>
      </fieldset>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="note" className="text-sm font-medium text-slate-700">
          Note <span className="font-normal text-slate-400">(optional)</span>
        </label>
        <input
          id="note"
          name="note"
          type="text"
          maxLength={1000}
          placeholder="Anything to add?"
          className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        />
      </div>

      {state?.error && (
        <p
          role="alert"
          className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
        >
          {state.error}
        </p>
      )}

      <button
        type="submit"
        disabled={pending}
        className="rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
      >
        {pending ? "Adding..." : "Add Expense"}
      </button>
    </form>
  );
}
