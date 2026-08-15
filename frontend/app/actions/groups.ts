"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";

import { apiFetch } from "@/lib/server-api";
import { getDict } from "@/lib/i18n";
import type { GroupSummary, Invitation } from "@/lib/api";

export type ActionState = { error?: string } | undefined;
export type InviteActionState = { error?: string; token?: string } | undefined;

async function errorMessage(err: unknown): Promise<ActionState> {
  const dict = await getDict();
  if (err instanceof Error) {
    return { error: err.message };
  }
  return { error: dict.errors.somethingWentWrong };
}

export async function createGroup(
  _state: ActionState,
  formData: FormData,
): Promise<ActionState> {
  const payload = new FormData();
  payload.set("name", String(formData.get("name") ?? "").trim());
  payload.set("currency", String(formData.get("currency") ?? "IDR").toUpperCase());
  const description = String(formData.get("description") ?? "").trim();
  if (description) payload.set("description", description);

  const logo = formData.get("logo");
  if (logo instanceof File && logo.size > 0) {
    payload.set("logo", logo, logo.name);
  }

  let groupId = "";
  try {
    const { group } = await apiFetch<{ group: GroupSummary }>("/api/v1/groups", {
      method: "POST",
      body: payload,
    });
    groupId = group.id;
  } catch (err) {
    return errorMessage(err);
  }

  revalidatePath("/groups");
  revalidatePath("/");
  redirect(`/groups/${groupId}?success=group-created`);
}

export async function inviteMember(
  groupId: string,
  _state: InviteActionState,
  formData: FormData,
): Promise<InviteActionState> {
  const email = String(formData.get("email") ?? "").trim();

  try {
    const { invitation } = await apiFetch<{ invitation: Invitation }>(
      `/api/v1/groups/${groupId}/invitations`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      },
    );
    return { token: invitation.token };
  } catch (err) {
    const dict = await getDict();
    if (err instanceof Error) {
      return { error: err.message };
    }
    return { error: dict.errors.somethingWentWrong };
  }
}

export async function acceptInvitation(
  _state: ActionState,
  formData: FormData,
): Promise<ActionState> {
  const token = String(formData.get("token") ?? "").trim();

  if (!token) {
    const dict = await getDict();
    return { error: dict.errors.tokenRequired };
  }

  let groupId = "";
  try {
    const { group } = await apiFetch<{ group: GroupSummary }>(
      `/api/v1/groups/invitations/${encodeURIComponent(token)}/accept`,
      { method: "POST" },
    );
    groupId = group.id;
  } catch (err) {
    return errorMessage(err);
  }

  revalidatePath("/groups");
  revalidatePath("/");
  redirect(`/groups/${groupId}?success=group-joined`);
}

export async function deleteGroup(groupId: string): Promise<ActionState> {
  try {
    await apiFetch(`/api/v1/groups/${groupId}`, { method: "DELETE" });
  } catch (err) {
    return errorMessage(err);
  }

  revalidatePath("/groups");
  revalidatePath("/");
  redirect("/groups?success=group-deleted");
}
