import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { API_URL, type User } from "./api";

export async function getCurrentUser(): Promise<User> {
  const cookieStore = await cookies();

  const res = await fetch(`${API_URL}/api/v1/me`, {
    headers: { Cookie: cookieStore.toString() },
    cache: "no-store",
  });

  if (!res.ok) {
    redirect("/login");
  }

  const body = (await res.json()) as { data: { user: User } };
  return body.data.user;
}
