"use client";

import { useActionState } from "react";
import { HandCoins } from "lucide-react";

import { createSettlement } from "@/app/actions/settlements";
import { formatCurrency } from "@/lib/format";
import type { Member, Suggestion } from "@/lib/api";

function SettleAction({ groupId, receiverId, amount }: { groupId: string; receiverId: string; amount?: string }) {
  const [state, action, pending] = useActionState(
    createSettlement.bind(null, groupId),
    undefined,
  );

  return (
    <form action={action} className="flex flex-col gap-3">
      <input type="hidden" name="receiverId" value={receiverId} />
      <input type="hidden" name="amount" value={amount ?? ""} />
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
        className="inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
      >
        <HandCoins className="h-4 w-4" aria-hidden="true" />
        {pending ? "Recording..." : "Record payment"}
      </button>
    </form>
  );
}

function SettleForm({ groupId, others }: { groupId: string; others: Member[] }) {
  const [state, action, pending] = useActionState(
    createSettlement.bind(null, groupId),
    undefined,
  );

  return (
    <form action={action} className="mt-2 flex flex-col gap-3">
      <div className="flex flex-col gap-1.5">
        <label htmlFor="settle-receiver" className="text-sm font-medium text-slate-700">
          I paid back
        </label>
        <select
          id="settle-receiver"
          name="receiverId"
          required
          defaultValue=""
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        >
          <option value="" disabled>
            Select a member
          </option>
          {others.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="settle-amount" className="text-sm font-medium text-slate-700">
          Amount
        </label>
        <input
          id="settle-amount"
          name="amount"
          type="text"
          required
          inputMode="decimal"
          placeholder="0.00"
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
        className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:opacity-60"
      >
        {pending ? "Recording..." : "Record payment"}
      </button>
    </form>
  );
}

export default function SettlePanel({
  groupId,
  members,
  myUserId,
  suggestions,
}: {
  groupId: string;
  members: Member[];
  myUserId: string;
  suggestions: Suggestion[];
}) {
  const others = members.filter((m) => m.id !== myUserId);
  const mySuggestions = suggestions.filter((s) => s.fromUserId === myUserId);

  return (
    <div className="flex flex-col gap-4">
      {mySuggestions.length > 0 && (
        <div>
          <p className="text-sm font-medium text-slate-700">Quick settle</p>
          <ul className="mt-2 flex flex-col">
            {mySuggestions.map((s) => {
              const receiver = members.find((m) => m.id === s.toUserId);
              return (
                <li
                  key={`${s.toUserId}-${s.amount}`}
                  className="flex items-center justify-between gap-3 border-t border-slate-100 py-3"
                >
                  <div className="text-sm">
                    <p className="font-medium text-slate-900">
                      You owe {receiver?.name ?? "a member"}
                    </p>
                    <p className="text-slate-500">{formatCurrency(s.amount)}</p>
                  </div>
                  <SettleAction groupId={groupId} receiverId={s.toUserId} amount={s.amount} />
                </li>
              );
            })}
          </ul>
        </div>
      )}

      <div>
        <p className="text-sm font-medium text-slate-700">Record a payment</p>
        <SettleForm groupId={groupId} others={others} />
      </div>
    </div>
  );
}
