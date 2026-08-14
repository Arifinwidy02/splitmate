"use client";

import { useActionState } from "react";

import { logout } from "@/app/actions/auth";

export default function LogoutButton() {
  const [, action, pending] = useActionState(logout, undefined);

  return (
    <form action={action}>
      <button
        type="submit"
        disabled={pending}
        className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:opacity-60"
      >
        {pending ? "Signing out..." : "Sign out"}
      </button>
    </form>
  );
}
