"use client";

import { useActionState, useState } from "react";
import { Check, Copy, Send } from "lucide-react";

import { inviteMembers } from "@/app/actions/groups";
import type { Dict } from "@/lib/i18n/id";

const failureReasons = {
  MEMBER_EXISTS: "inviteFailureMemberExists",
  INVITATION_EXISTS: "inviteFailureInvitationExists",
  DUPLICATE: "inviteFailureDuplicate",
} as const;

export default function InviteForm({ groupId, dict }: { groupId: string; dict: Dict }) {
  const [state, action, pending] = useActionState(
    inviteMembers.bind(null, groupId),
    undefined,
  );
  const [copiedAll, setCopiedAll] = useState(false);
  const [copiedEmail, setCopiedEmail] = useState<string | null>(null);

  const copyToken = async (email: string, token: string) => {
    await navigator.clipboard.writeText(token);
    setCopiedEmail(email);
    setTimeout(() => setCopiedEmail(null), 2000);
  };

  const copyAll = async () => {
    if (!state?.invitations?.length) return;
    const text = state.invitations
      .map((inv) => `${inv.email}: ${inv.token}`)
      .join("\n");
    await navigator.clipboard.writeText(text);
    setCopiedAll(true);
    setTimeout(() => setCopiedAll(false), 2000);
  };

  const failureText = (reason: string) =>
    dict.groups[failureReasons[reason as keyof typeof failureReasons] ?? "inviteFailureUnknown"];

  return (
    <div className="flex flex-col gap-3">
      <form action={action} className="flex flex-col gap-2 sm:flex-row sm:items-start">
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <label htmlFor="invite-emails" className="text-sm font-medium text-slate-700">
            {dict.groups.inviteEmailsLabel}
          </label>
          <textarea
            id="invite-emails"
            name="emails"
            rows={4}
            placeholder={dict.groups.inviteEmailsPlaceholder}
            className="w-full resize-y rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>
        <button
          type="submit"
          disabled={pending}
          className="inline-flex shrink-0 items-center gap-2 self-end rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60 sm:mt-6"
        >
          <Send className="h-4 w-4" aria-hidden="true" />
          {pending ? dict.groups.inviting : dict.groups.invite}
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

      {state?.invitations && state.invitations.length > 0 && (
        <div className="rounded-xl border border-green-200 bg-green-50 p-3">
          <div className="flex items-center justify-between gap-2">
            <p className="text-sm font-medium text-green-800">{dict.groups.inviteCreated}</p>
            <button
              type="button"
              onClick={copyAll}
              aria-label={dict.groups.copyAllAria}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-green-300 bg-white px-3 py-1.5 text-xs font-semibold text-green-800 transition hover:bg-green-100"
            >
              {copiedAll ? (
                <Check className="h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <Copy className="h-3.5 w-3.5" aria-hidden="true" />
              )}
              {copiedAll ? dict.groups.copied : dict.groups.copyAll}
            </button>
          </div>
          <ul className="mt-2 flex flex-col gap-2">
            {state.invitations.map((inv) => (
              <li key={inv.email} className="flex items-center gap-2">
                <span className="min-w-0 flex-1 truncate text-xs text-green-800">
                  {inv.email}
                </span>
                <code className="min-w-0 flex-[2] break-all rounded-lg bg-white px-3 py-1.5 font-mono text-xs text-slate-800">
                  {inv.token}
                </code>
                <button
                  type="button"
                  onClick={() => copyToken(inv.email, inv.token)}
                  aria-label={dict.groups.copyTokenAria}
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-green-300 bg-white px-3 py-1.5 text-xs font-semibold text-green-800 transition hover:bg-green-100"
                >
                  {copiedEmail === inv.email ? (
                    <Check className="h-3.5 w-3.5" aria-hidden="true" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" aria-hidden="true" />
                  )}
                  {copiedEmail === inv.email ? dict.groups.copied : dict.groups.copy}
                </button>
              </li>
            ))}
          </ul>
          <p className="mt-2 text-xs text-green-700">{dict.groups.tokenExpiry}</p>
        </div>
      )}

      {state?.failed && state.failed.length > 0 && (
        <div className="rounded-xl border border-red-200 bg-red-50 p-3">
          <p className="text-sm font-medium text-red-800">{dict.groups.inviteFailed}</p>
          <ul className="mt-2 flex flex-col gap-1.5">
            {state.failed.map((f) => (
              <li key={f.email} className="flex items-center gap-2 text-xs">
                <span className="min-w-0 flex-1 truncate text-red-800">{f.email}</span>
                <span className="shrink-0 text-red-600">{failureText(f.reason)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}