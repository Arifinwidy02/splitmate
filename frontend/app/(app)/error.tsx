"use client";

import { TriangleAlert } from "lucide-react";

import { dictionaries } from "@/lib/i18n/dictionaries";
import { getClientLocale } from "@/lib/i18n/constants";

export default function Error(props: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const { reset } = props;
  const dict = dictionaries[getClientLocale()];

  return (
    <div className="mx-auto flex w-full max-w-md flex-col items-center rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-card">
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-red-50">
        <TriangleAlert className="h-6 w-6 text-red-600" aria-hidden="true" />
      </span>
      <h1 className="mt-4 text-lg font-semibold text-slate-900">
        {dict.errors.pageErrorTitle}
      </h1>
      <p className="mt-1 text-sm text-slate-500">{dict.errors.pageErrorBody}</p>
      <button
        type="button"
        onClick={reset}
        className="mt-5 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700"
      >
        {dict.errors.tryAgain}
      </button>
    </div>
  );
}