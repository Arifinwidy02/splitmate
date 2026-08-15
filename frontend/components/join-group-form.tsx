"use client";

import { useActionState } from "react";
import { Ticket } from "lucide-react";

import { acceptInvitation } from "@/app/actions/groups";
import type { Dict } from "@/lib/i18n/id";

export default function JoinGroupForm({ dict }: { dict: Dict }) {
  const [state, action, pending] = useActionState(acceptInvitation, undefined);

  return (
    <form action={action} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <label htmlFor="token" className="text-sm font-medium text-slate-700">
          {dict.groups.tokenLabel}
        </label>
        <input
          id="token"
          name="token"
          type="text"
          required
          placeholder={dict.groups.tokenPlaceholder}
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
        className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:opacity-60"
      >
        <Ticket className="h-4 w-4" aria-hidden="true" />
        {pending ? dict.groups.joining : dict.groups.join}
      </button>
    </form>
  );
}
