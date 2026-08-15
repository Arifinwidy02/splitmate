"use client";

import { useEffect, useRef, useState } from "react";
import { TriangleAlert } from "lucide-react";

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel,
  pendingLabel,
  cancelLabel,
  pending,
  onConfirm,
  onClose,
}: {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  pendingLabel: string;
  cancelLabel: string;
  pending: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [supported] = useState(
    () =>
      typeof HTMLDialogElement !== "undefined" &&
      typeof HTMLDialogElement.prototype.showModal === "function",
  );

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && supported) {
      if (!dialog.open) dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open, supported]);

  if (!open || !supported) return null;

  return (
    <dialog
      ref={dialogRef}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      className="m-auto w-[calc(100%-2rem)] max-w-sm rounded-2xl border border-slate-200 bg-white p-5 shadow-xl backdrop:bg-slate-900/40"
      aria-labelledby="confirm-dialog-title"
      aria-describedby="confirm-dialog-message"
    >
      <div className="flex items-start gap-3">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-red-50">
          <TriangleAlert className="h-5 w-5 text-red-600" aria-hidden="true" />
        </span>
        <div>
          <h2 id="confirm-dialog-title" className="text-base font-semibold text-slate-900">
            {title}
          </h2>
          <p id="confirm-dialog-message" className="mt-1 text-sm text-slate-500">
            {message}
          </p>
        </div>
      </div>

      <div className="mt-5 flex justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          disabled={pending}
          className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:opacity-60"
        >
          {cancelLabel}
        </button>
        <button
          type="button"
          onClick={onConfirm}
          disabled={pending}
          className="rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-red-700 disabled:opacity-60"
        >
          {pending ? pendingLabel : confirmLabel}
        </button>
      </div>
    </dialog>
  );
}
