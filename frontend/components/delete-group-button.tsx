"use client";

import { useState } from "react";
import { Trash2 } from "lucide-react";

import ConfirmDialog from "@/components/confirm-dialog";
import { deleteGroup } from "@/app/actions/groups";

export default function DeleteGroupButton({
  groupId,
  groupName,
}: {
  groupId: string;
  groupName: string;
}) {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | undefined>();

  const onConfirm = async () => {
    setPending(true);
    setError(undefined);
    const result = await deleteGroup(groupId);
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
        className="inline-flex items-center gap-1.5 rounded-lg border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition hover:bg-red-50"
      >
        <Trash2 className="h-4 w-4" aria-hidden="true" />
        Delete group
      </button>

      <ConfirmDialog
        open={open}
        title="Delete this group?"
        message={`"${groupName}" and all its expenses will be permanently deleted. This cannot be undone.`}
        confirmLabel="Delete group"
        pending={pending}
        onConfirm={onConfirm}
        onClose={() => setOpen(false)}
      />

      {error && (
        <p role="alert" className="mt-1 text-right text-sm text-red-600">
          {error}
        </p>
      )}
    </div>
  );
}
