"use client";

import Link from "next/link";
import { useActionState } from "react";

import { register, type AuthActionState } from "@/app/actions/auth";
import GoogleButton from "@/components/google-button";
import type { Dict } from "@/lib/i18n/id";

export default function RegisterForm({ dict }: { dict: Dict }) {
  const [state, action, pending] = useActionState<AuthActionState, FormData>(
    register,
    undefined,
  );

  return (
    <>
      <form action={action} className="mt-6 flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <label htmlFor="name" className="text-sm font-medium text-slate-700">
            {dict.auth.name}
          </label>
          <input
            id="name"
            name="name"
            type="text"
            required
            autoComplete="name"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="email" className="text-sm font-medium text-slate-700">
            {dict.auth.email}
          </label>
          <input
            id="email"
            name="email"
            type="email"
            required
            autoComplete="email"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label
            htmlFor="password"
            className="text-sm font-medium text-slate-700"
          >
            {dict.auth.password}
          </label>
          <input
            id="password"
            name="password"
            type="password"
            required
            minLength={8}
            autoComplete="new-password"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
          <p className="text-xs text-slate-500">{dict.auth.passwordHint}</p>
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
          className="mt-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
        >
          {pending ? dict.auth.creatingAccount : dict.auth.createAccountBtn}
        </button>

        <p className="text-center text-sm text-slate-500">
          {dict.auth.alreadyAccount}{" "}
          <Link
            href="/login"
            className="font-medium text-green-600 hover:underline"
          >
            {dict.auth.signInLink}
          </Link>
        </p>
      </form>
      <div className="my-4 flex items-center gap-3 text-xs text-slate-400">
        <span className="h-px flex-1 bg-slate-200" />
        {dict.common.or}
        <span className="h-px flex-1 bg-slate-200" />
      </div>
      <GoogleButton label={dict.auth.continueWithGoogle} />
    </>
  );
}
