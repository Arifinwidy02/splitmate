"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { API_URL } from "@/lib/api";
import { getDict } from "@/lib/i18n";

export type AuthActionState = { error?: string } | undefined;

const SESSION_COOKIE = "session";
const SESSION_MAX_AGE = 7 * 24 * 60 * 60;

async function storeSessionFromResponse(res: Response) {
  const sessionCookie = res.headers
    .getSetCookie()
    .find((cookie) => cookie.startsWith(`${SESSION_COOKIE}=`));
  if (!sessionCookie) return;

  const value = sessionCookie
    .slice(`${SESSION_COOKIE}=`.length)
    .split(";")[0];

  const cookieStore = await cookies();
  cookieStore.set(SESSION_COOKIE, value, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    maxAge: SESSION_MAX_AGE,
    path: "/",
  });
}

async function errorMessage(res: Response, fallback: string): Promise<string> {
  const body = await res.json().catch(() => null);
  return body?.error?.message ?? fallback;
}

export async function login(
  _state: AuthActionState,
  formData: FormData,
): Promise<AuthActionState> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email: formData.get("email"),
      password: formData.get("password"),
    }),
    cache: "no-store",
  });

  if (!res.ok) {
    const dict = await getDict();
    return { error: await errorMessage(res, dict.errors.signInFailed) };
  }

  await storeSessionFromResponse(res);
  redirect("/");
}

export async function register(
  _state: AuthActionState,
  formData: FormData,
): Promise<AuthActionState> {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: formData.get("name"),
      email: formData.get("email"),
      password: formData.get("password"),
    }),
    cache: "no-store",
  });

  if (!res.ok) {
    const dict = await getDict();
    return {
      error: await errorMessage(res, dict.errors.registrationFailed),
    };
  }

  redirect("/login");
}

export async function logout() {
  const cookieStore = await cookies();

  await fetch(`${API_URL}/api/v1/auth/logout`, {
    method: "POST",
    headers: { Cookie: cookieStore.toString() },
    cache: "no-store",
  });

  cookieStore.delete(SESSION_COOKIE);
  redirect("/login");
}
