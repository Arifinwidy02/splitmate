"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { API_URL } from "@/lib/api";
import { getDict } from "@/lib/i18n";
import {
  ACCESS_TOKEN_COOKIE,
  ACCESS_TOKEN_MAX_AGE,
  REFRESH_TOKEN_COOKIE,
  REFRESH_TOKEN_MAX_AGE,
  extractTokenFromSetCookie,
} from "@/lib/tokens";

export type AuthActionState = { error?: string } | undefined;

function getNextUrl(formData: FormData): string {
  const next = formData.get("next");
  if (typeof next === "string" && next.startsWith("/")) {
    return next;
  }
  return "/dashboard";
}

async function storeTokensFromResponse(res: Response) {
  const setCookies = res.headers.getSetCookie();
  const cookieStore = await cookies();

  const accessToken = setCookies
    .map((c) => extractTokenFromSetCookie(c, ACCESS_TOKEN_COOKIE))
    .find((v) => v !== null);

  const refreshToken = setCookies
    .map((c) => extractTokenFromSetCookie(c, REFRESH_TOKEN_COOKIE))
    .find((v) => v !== null);

  if (accessToken) {
    cookieStore.set(ACCESS_TOKEN_COOKIE, accessToken, {
      httpOnly: true,
      sameSite: "lax",
      secure: process.env.NODE_ENV === "production",
      maxAge: ACCESS_TOKEN_MAX_AGE,
      path: "/",
    });
  }

  if (refreshToken) {
    cookieStore.set(REFRESH_TOKEN_COOKIE, refreshToken, {
      httpOnly: true,
      sameSite: "strict",
      secure: process.env.NODE_ENV === "production",
      maxAge: REFRESH_TOKEN_MAX_AGE,
      path: "/",
    });
  }
}

async function clearAuthCookies() {
  const cookieStore = await cookies();
  cookieStore.delete(ACCESS_TOKEN_COOKIE);
  cookieStore.delete(REFRESH_TOKEN_COOKIE);
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

  await storeTokensFromResponse(res);
  const nextUrl = getNextUrl(formData);
  redirect(nextUrl);
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

  const nextUrl = getNextUrl(formData);
  redirect(nextUrl);
}

export async function logout() {
  const cookieStore = await cookies();

  await fetch(`${API_URL}/api/v1/auth/logout`, {
    method: "POST",
    headers: { Cookie: cookieStore.toString() },
    cache: "no-store",
  });

  await clearAuthCookies();
  redirect("/login");
}
