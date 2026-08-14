"use client";

import { useActionState, useState } from "react";
import { Check, Copy, Send } from "lucide-react";

import { inviteMember } from "@/app/actions/groups";

export default function InviteForm({ groupId }: { groupId: string }) {
  const [state, action, pending] = useActionState(
    inviteMember.bind(null, groupId),
    undefined,
  );
  const [copied, setCopied] = useState(false);

  const copyToken = async () => {
    if (!state?.token) return;
    await navigator.clipboard.writeText(state.token);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex flex-col gap-3">
      <form action={action} className="flex gap-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <label htmlFor="invite-email" className="text-sm font-medium text-slate-700">
            Email
          </label>
          <input
            id="invite-email"
            name="email"
            type="email"
            required
            placeholder="friend@example.com"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>
        <button
          type="submit"
          disabled={pending}
          className="mt-6 inline-flex items-center gap-2 self-start rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
        >
          <Send className="h-4 w-4" aria-hidden="true" />
          {pending ? "Sending..." : "Invite"}
        </button>
      </form>

      {state?.error && (
        <p
          role="alert"
          className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
        >
          {state.error}
        </p>
      )}

      {state?.token && (
        <div className="rounded-xl border border-green-200 bg-green-50 p-3">
          <p className="text-sm font-medium text-green-800">
            Invitation created — share this token with your friend.
          </p>
          <div className="mt-2 flex items-center gap-2">
            <code className="min-w-0 flex-1 break-all rounded-lg bg-white px-3 py-2 text-xs text-slate-800">
              {state.token}
            </code>
            <button
              type="button"
              onClick={copyToken}
              aria-label="Copy invitation token"
              className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-green-300 bg-white px-3 py-2 text-xs font-semibold text-green-800 transition hover:bg-green-100"
            >
              {copied ? (
                <Check className="h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <Copy className="h-3.5 w-3.5" aria-hidden="true" />
              )}
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
          <p className="mt-2 text-xs text-green-700">
            The token is shown only once and expires in 7 days.
          </p>
        </div>
      )}
    </div>
  );
}
