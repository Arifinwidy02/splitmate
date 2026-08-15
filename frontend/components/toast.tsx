"use client";

import { useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import type { Dict } from "@/lib/i18n/id";

export default function Toast({ success, dict }: { success?: string; dict: Dict }) {
  const router = useRouter();
  const shown = useRef(false);

  useEffect(() => {
    if (!success || shown.current) return;
    shown.current = true;

    const message = (dict.toast as Record<string, string>)[success];
    if (!message) return;

    toast.success(message);

    const url = new URL(window.location.href);
    url.searchParams.delete("success");
    router.replace(url.pathname, { scroll: false });
  }, [success, dict, router]);

  return null;
}