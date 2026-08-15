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

  const expenseDate =
    String(formData.get("expenseDateRfc") ?? "") ||
    toRFC3339(String(formData.get("expenseDate") ?? ""));

  const payload = new FormData();
  payload.set("description", String(formData.get("description") ?? "").trim());
  payload.set("amount", String(formData.get("amount") ?? ""));
  payload.set("currency", String(formData.get("currency") ?? "IDR"));
  payload.set("paidBy", String(formData.get("paidBy") ?? ""));
  payload.set("category", String(formData.get("category") ?? "Other"));
  payload.set("expenseDate", expenseDate);
  payload.set("splitType", splitType);
  const note = String(formData.get("note") ?? "").trim();
  if (note) payload.set("note", note);

  for (const id of participants) {
    payload.append("participant", id);
    if (splitType === "custom") {
      payload.set(`split.${id}`, String(formData.get(`split-${id}`) ?? ""));
    }
  }

  const receipt = formData.get("receipt");
  if (receipt instanceof File && receipt.size > 0) {
    payload.set("receipt", receipt, receipt.name);
  }

  try {
    await apiFetch<{ expense: ExpenseDetail }>(
      `/api/v1/groups/${groupId}/expenses`,
      {
        method: "POST",
        body: payload,
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