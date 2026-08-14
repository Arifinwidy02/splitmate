"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";

import { apiFetch } from "@/lib/server-api";
import { getCurrentUser } from "@/lib/auth";
import type { Settlement, User } from "@/lib/api";

export type ActionState = { error?: string } | undefined;

async function errorMessage(err: unknown): Promise<ActionState> {
  if (err instanceof Error) {
    return { error: err.message };
  }
  return { error: "Something went wrong. Please try again." };
}

export async function createSettlement(
  groupId: string,
  _state: ActionState,
  formData: FormData,
): Promise<ActionState> {
  const user: User = await getCurrentUser();

  const amount = String(formData.get("amount") ?? "");
  const receiverId = String(formData.get("receiverId") ?? "");
  const settledAt = String(formData.get("settledAt") ?? "").trim();

  try {
    await apiFetch<Settlement>(`/api/v1/groups/${groupId}/settlements`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        payerId: user.id,
        receiverId,
        amount,
        settledAt: settledAt || undefined,
      }),
    });
  } catch (err) {
    return errorMessage(err);
  }

  revalidatePath(`/groups/${groupId}`);
  revalidatePath("/");
  redirect(`/groups/${groupId}?success=settlement-recorded`);
}