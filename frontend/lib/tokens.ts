export const ACCESS_TOKEN_COOKIE = "access_token";
export const REFRESH_TOKEN_COOKIE = "refresh_token";
export const ACCESS_TOKEN_MAX_AGE = 15 * 60; // 15 minutes, matches backend AccessTokenTTL
export const REFRESH_TOKEN_MAX_AGE = 7 * 24 * 60 * 60; // 7 days, matches backend RefreshTokenTTL

/**
 * Cheap expiry check based on the JWT payload only.
 * The backend is still the source of truth for signature validation.
 */
export function isAccessTokenExpired(token: string): boolean {
  try {
    const payload = JSON.parse(
      new TextDecoder().decode(
        Uint8Array.from(
          atob(token.split(".")[1]!.replace(/-/g, "+").replace(/_/g, "/")),
          (c) => c.charCodeAt(0),
        ),
      ),
    );
    return typeof payload.exp !== "number" || payload.exp * 1000 <= Date.now();
  } catch {
    return true;
  }
}

export function extractTokenFromSetCookie(
  setCookie: string,
  cookieName: string,
): string | null {
  if (!setCookie.startsWith(`${cookieName}=`)) return null;
  const value = setCookie.slice(cookieName.length + 1).split(";")[0] ?? "";
  return value.length > 0 ? value : null;
}
