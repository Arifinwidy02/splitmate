import Link from "next/link";
import { SearchX } from "lucide-react";

import { getDict } from "@/lib/i18n";

export default async function NotFound() {
  const dict = await getDict();

  return (
    <main className="flex min-h-full flex-1 items-center justify-center bg-slate-50 p-8">
      <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-card">
        <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
          <SearchX className="h-6 w-6 text-slate-500" aria-hidden="true" />
        </span>
        <h1 className="mt-4 text-xl font-bold text-slate-900">{dict.notFound.title}</h1>
        <p className="mt-1 text-sm text-slate-500">{dict.notFound.body}</p>
        <Link
          href="/dashboard"
          className="mt-5 inline-block rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700"
        >
          {dict.notFound.backToDashboard}
        </Link>
      </div>
    </main>
  );
}