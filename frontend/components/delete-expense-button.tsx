"use client";

import { useState } from "react";
import { Trash2 } from "lucide-react";

import ConfirmDialog from "@/components/confirm-dialog";
import { deleteExpense } from "@/app/actions/expenses";
import type { Dict } from "@/lib/i18n/id";
import { tr } from "@/lib/i18n/tr";

export default function DeleteExpenseButton({
  groupId,
  expenseId,
  description,
  dict,
}: {
  groupId: string;
  expenseId: string;
  description: string;
  dict: Dict;
}) {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | undefined>();

  const onConfirm = async () => {
    setPending(true);
    setError(undefined);
    const result = await deleteExpense(groupId, expenseId);
    if (result?.error) {
      setError(result.error);
      setPending(false);
    }
  };

  return (
    <div className="flex flex-col items-end">
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={`${dict.group.deleteExpense} ${description}`}
        className="rounded-lg p-1.5 text-slate-400 transition hover:bg-red-50 hover:text-red-600"
      >
        <Trash2 className="h-4 w-4" aria-hidden="true" />
      </button>

      <ConfirmDialog
        open={open}
        title={dict.group.deleteExpenseTitle}
        message={tr(dict.group.deleteExpenseMessage, { description })}
        confirmLabel={dict.group.deleteExpense}
        pendingLabel={dict.common.deleting}
        cancelLabel={dict.common.cancel}
        pending={pending}
        onConfirm={onConfirm}
        onClose={() => setOpen(false)}
      />

      {error && (
        <p role="alert" className="mt-1 text-right text-xs text-red-600">
          {error}
        </p>
      )}
    </div>
  );
}
