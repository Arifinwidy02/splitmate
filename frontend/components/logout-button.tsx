"use client";

import { useActionState } from "react";

import { logout } from "@/app/actions/auth";
import type { Dict } from "@/lib/i18n/id";

export default function LogoutButton({ dict }: { dict: Dict }) {
  const [, action, pending] = useActionState(logout, undefined);

  return (
    <form action={action}>
      <button
        type="submit"
        disabled={pending}
        className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:opacity-60"
      >
        {pending ? dict.auth.signingOut : dict.auth.logout}
      </button>
    </form>
  );
}
