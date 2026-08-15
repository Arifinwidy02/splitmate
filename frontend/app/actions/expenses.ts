"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";

import { apiFetch } from "@/lib/server-api";
import { toRFC3339 } from "@/lib/format";
import { getDict } from "@/lib/i18n";
import type { ExpenseDetail } from "@/lib/api";

export type ActionState = { error?: string } | undefined;

async function errorMessage(err: unknown): Promise<ActionState> {
  const dict = await getDict();
  if (err instanceof Error) {
    return { error: err.message };
  }
  return { error: dict.errors.somethingWentWrong };
}

export async function createExpense(
  groupId: string,
  _state: ActionState,
  formData: FormData,
): Promise<ActionState> {
  const splitType = String(formData.get("splitType") ?? "equal");
  const participants = formData
    .getAll("participant")
    .map((v) => String(v));

  const splits =
    splitType === "custom"
      ? participants.map((userId) => ({
          userId,
          amount: String(formData.get(`split-${userId}`) ?? ""),
        }))
      : undefined;

  const expenseDate =
    String(formData.get("expenseDateRfc") ?? "") ||
    toRFC3339(String(formData.get("expenseDate") ?? ""));

  const payload = {
    description: String(formData.get("description") ?? "").trim(),
    amount: String(formData.get("amount") ?? ""),
    currency: String(formData.get("currency") ?? "IDR"),
    paidBy: String(formData.get("paidBy") ?? ""),
    category: String(formData.get("category") ?? "Other"),
    expenseDate,
    note: String(formData.get("note") ?? "").trim() || undefined,
    splitType,
    participants: splitType === "equal" ? participants : undefined,
    splits,
  };

  try {
    await apiFetch<{ expense: ExpenseDetail }>(
      `/api/v1/groups/${groupId}/expenses`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      },
    );
  } catch (err) {
    return errorMessage(err);
  }

  revalidatePath(`/groups/${groupId}`);
  revalidatePath("/");
  redirect(`/groups/${groupId}?success=expense-added`);
}

export async function deleteExpense(
  groupId: string,
  expenseId: string,
): Promise<ActionState> {
  try {
    await apiFetch(`/api/v1/expenses/${expenseId}`, { method: "DELETE" });
  } catch (err) {
    return errorMessage(err);
  }

  revalidatePath(`/groups/${groupId}`);
  revalidatePath("/");
  redirect(`/groups/${groupId}?success=expense-deleted`);
}