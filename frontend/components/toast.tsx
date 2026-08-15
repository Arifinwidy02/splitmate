"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { CheckCircle2, X } from "lucide-react";

import type { Dict } from "@/lib/i18n/id";

export default function Toast({ success, dict }: { success?: string; dict: Dict }) {
  const router = useRouter();
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    if (!success) return;
    const timer = setTimeout(() => setDismissed(true), 4000);
    return () => clearTimeout(timer);
  }, [success]);

  useEffect(() => {
    if (!success || !dismissed) return;
    const cleanTimer = setTimeout(() => {
      const url = new URL(window.location.href);
      url.searchParams.delete("success");
      router.replace(url.pathname, { scroll: false });
    }, 300);
    return () => clearTimeout(cleanTimer);
  }, [success, dismissed, router]);

  const message = success
  ? (dict.toast as Record<string, string>)[success]
  : undefined;
  if (!message || dismissed) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="fixed bottom-20 right-4 z-50 flex items-center gap-3 rounded-xl border border-green-200 bg-white px-4 py-3 shadow-lg lg:bottom-6 lg:right-6"
    >
      <CheckCircle2 className="h-5 w-5 shrink-0 text-green-600" aria-hidden="true" />
      <p className="text-sm font-medium text-slate-800">{message}</p>
      <button
        type="button"
        onClick={() => setDismissed(true)}
        aria-label={dict.toast.dismiss}
        className="ml-2 rounded-md p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-600"
      >
        <X className="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
  );
}
