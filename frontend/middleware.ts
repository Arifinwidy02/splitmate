import { NextRequest, NextResponse } from "next/server";

import { API_URL } from "@/lib/api";
import {
  ACCESS_TOKEN_COOKIE,
  ACCESS_TOKEN_MAX_AGE,
  REFRESH_TOKEN_COOKIE,
  REFRESH_TOKEN_MAX_AGE,
  extractTokenFromSetCookie,
  isAccessTokenExpired,
} from "@/lib/tokens";

function cookieAttributes(name: string) {
  return {
    httpOnly: true,
    sameSite: name === REFRESH_TOKEN_COOKIE ? ("strict" as const) : ("lax" as const),
    secure: process.env.NODE_ENV === "production",
    maxAge: name === ACCESS_TOKEN_COOKIE ? ACCESS_TOKEN_MAX_AGE : REFRESH_TOKEN_MAX_AGE,
    path: "/",
  };
}

// Next 16 renamed middleware.ts -> proxy.ts, but proxy.ts is not executed
// on Vercel production builds (vercel/next.js#86241, #86303). Keep the
// deprecated middleware.ts convention so the silent refresh runs everywhere.
export async function middleware(request: NextRequest) {
  const accessToken = request.cookies.get(ACCESS_TOKEN_COOKIE)?.value;
  const refreshToken = request.cookies.get(REFRESH_TOKEN_COOKIE)?.value;

  if (accessToken && !isAccessTokenExpired(accessToken)) {
    return NextResponse.next();
  }

  if (!refreshToken) {
    return NextResponse.next();
  }

  let res: Response;
  try {
    res = await fetch(`${API_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Cookie: `${REFRESH_TOKEN_COOKIE}=${refreshToken}`,
      },
      cache: "no-store",
    });
  } catch {
    return NextResponse.next();
  }

  if (!res.ok) {
    return NextResponse.next();
  }

  const setCookies = res.headers.getSetCookie();
  const newAccessToken = setCookies
    .map((c) => extractTokenFromSetCookie(c, ACCESS_TOKEN_COOKIE))
    .find((v) => v !== null);
  const newRefreshToken = setCookies
    .map((c) => extractTokenFromSetCookie(c, REFRESH_TOKEN_COOKIE))
    .find((v) => v !== null);

  if (!newAccessToken) {
    return NextResponse.next();
  }

  request.cookies.set(ACCESS_TOKEN_COOKIE, newAccessToken);
  if (newRefreshToken) {
    request.cookies.set(REFRESH_TOKEN_COOKIE, newRefreshToken);
  }

  const response = NextResponse.next({ request: { headers: request.headers } });

  response.cookies.set(ACCESS_TOKEN_COOKIE, newAccessToken, cookieAttributes(ACCESS_TOKEN_COOKIE));
  if (newRefreshToken) {
    response.cookies.set(REFRESH_TOKEN_COOKIE, newRefreshToken, cookieAttributes(REFRESH_TOKEN_COOKIE));
  }

  return response;
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon\\.ico|api/v1/auth/refresh|.*\\.(?:png|jpg|jpeg|svg|ico|webp)$).*)",
  ],
};
