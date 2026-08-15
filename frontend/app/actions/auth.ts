"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { API_URL } from "@/lib/api";
import { getDict } from "@/lib/i18n";

export type AuthActionState = { error?: string } | undefined;

const ACCESS_TOKEN_COOKIE = "access_token";
const REFRESH_TOKEN_COOKIE = "refresh_token";
const ACCESS_TOKEN_MAX_AGE = 15 * 60; // 15 minutes
const REFRESH_TOKEN_MAX_AGE = 7 * 24 * 60 * 60; // 7 days

function extractCookieValue(
  setCookie: string,
  cookieName: string,
): string | null {
  if (!setCookie.startsWith(`${cookieName}=`)) return null;
  const value = setCookie.slice(`${cookieName}=`.length).split(";")[0];
  return value;
}

async function storeTokensFromResponse(res: Response) {
  const setCookies = res.headers.getSetCookie();
  const cookieStore = await cookies();

  const accessToken = setCookies
    .map((c) => extractCookieValue(c, ACCESS_TOKEN_COOKIE))
    .find((v) => v !== null);

  const refreshToken = setCookies
    .map((c) => extractCookieValue(c, REFRESH_TOKEN_COOKIE))
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
  redirect("/?success=signed-in");
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

  redirect("/login?success=registered");
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
