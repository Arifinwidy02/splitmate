"use client";

import { useActionState } from "react";

import { createGroup } from "@/app/actions/groups";
import type { Dict } from "@/lib/i18n/id";

export default function CreateGroupForm({ dict }: { dict: Dict }) {
  const [state, action, pending] = useActionState(createGroup, undefined);

  return (
    <form action={action} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <label htmlFor="name" className="text-sm font-medium text-slate-700">
          {dict.groups.groupName}
        </label>
        <input
          id="name"
          name="name"
          type="text"
          required
          maxLength={100}
          placeholder={dict.groups.groupNamePlaceholder}
          className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="description" className="text-sm font-medium text-slate-700">
          {dict.groups.description}{" "}
          <span className="font-normal text-slate-400">{dict.common.optional}</span>
        </label>
        <input
          id="description"
          name="description"
          type="text"
          maxLength={500}
          placeholder={dict.groups.descriptionPlaceholder}
          className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="currency" className="text-sm font-medium text-slate-700">
          {dict.groups.currency}
        </label>
        <select
          id="currency"
          name="currency"
          defaultValue="IDR"
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        >
          <option value="IDR">{dict.groups.currencyIdr}</option>
          <option value="USD">{dict.groups.currencyUsd}</option>
          <option value="EUR">{dict.groups.currencyEur}</option>
          <option value="SGD">{dict.groups.currencySgd}</option>
        </select>
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
        {pending ? dict.groups.creating : dict.groups.create}
      </button>
    </form>
  );
}
