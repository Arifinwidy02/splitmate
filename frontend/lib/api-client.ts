import { API_URL } from "./api";

const ACCESS_TOKEN_COOKIE = "access_token";
const REFRESH_TOKEN_COOKIE = "refresh_token";

interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
}

let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

function subscribeTokenRefresh(callback: (token: string) => void) {
  refreshSubscribers.push(callback);
}

function onTokenRefreshed(token: string) {
  refreshSubscribers.forEach((callback) => callback(token));
  refreshSubscribers = [];
}

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;

  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) {
    return parts.pop()?.split(";").shift() || null;
  }
  return null;
}

function setCookie(name: string, value: string, maxAge: number) {
  if (typeof document === "undefined") return;

  document.cookie = `${name}=${value}; path=/; max-age=${maxAge}; SameSite=${
    name === REFRESH_TOKEN_COOKIE ? "Strict" : "Lax"
  }; ${process.env.NODE_ENV === "production" ? "Secure" : ""}`;
}

function clearCookie(name: string) {
  if (typeof document === "undefined") return;

  document.cookie = `${name}=; path=/; max-age=0; SameSite=${
    name === REFRESH_TOKEN_COOKIE ? "Strict" : "Lax"
  }; ${process.env.NODE_ENV === "production" ? "Secure" : ""}`;
}

async function refreshAccessToken(): Promise<string> {
  const refreshToken = getCookie(REFRESH_TOKEN_COOKIE);
  if (!refreshToken) {
    throw new Error("No refresh token available");
  }

  const res = await fetch(`${API_URL}/api/v1/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    // Clear cookies on refresh failure
    clearCookie(ACCESS_TOKEN_COOKIE);
    clearCookie(REFRESH_TOKEN_COOKIE);
    throw new Error("Failed to refresh token");
  }

  const setCookies = res.headers.getSetCookie();

  const accessToken = setCookies
    .map((c) => {
      if (c.startsWith(`${ACCESS_TOKEN_COOKIE}=`)) {
        return c.slice(`${ACCESS_TOKEN_COOKIE}=`.length).split(";")[0];
      }
      return null;
    })
    .find((v) => v !== null);

  const newRefreshToken = setCookies
    .map((c) => {
      if (c.startsWith(`${REFRESH_TOKEN_COOKIE}=`)) {
        return c.slice(`${REFRESH_TOKEN_COOKIE}=`.length).split(";")[0];
      }
      return null;
    })
    .find((v) => v !== null);

  if (accessToken) {
    setCookie(ACCESS_TOKEN_COOKIE, accessToken, 15 * 60); // 15 minutes
  }

  if (newRefreshToken) {
    setCookie(REFRESH_TOKEN_COOKIE, newRefreshToken, 7 * 24 * 60 * 60); // 7 days
  }

  if (!accessToken) {
    throw new Error("No access token in refresh response");
  }

  return accessToken;
}

export async function apiFetch(
  url: string,
  options: FetchOptions = {},
): Promise<Response> {
  const { skipAuth = false, ...fetchOptions } = options;

  let accessToken = getCookie(ACCESS_TOKEN_COOKIE);

  const makeRequest = async (token: string | null): Promise<Response> => {
    const headers: Record<string, string> = {
      ...(fetchOptions.headers as Record<string, string>),
      "Content-Type": "application/json",
    };

    if (token && !skipAuth) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    return fetch(url, {
      ...fetchOptions,
      headers,
      credentials: "include",
    });
  };

  let response = await makeRequest(accessToken);

  // If we get a 401 and we have a refresh token, try to refresh
  if (response.status === 401 && !skipAuth && getCookie(REFRESH_TOKEN_COOKIE)) {
    if (isRefreshing) {
      // Wait for the current refresh to complete
      return new Promise((resolve, reject) => {
        subscribeTokenRefresh((token) => {
          makeRequest(token).then(resolve).catch(reject);
        });
      });
    }

    isRefreshing = true;
    try {
      const newToken = await refreshAccessToken();
      onTokenRefreshed(newToken);
      response = await makeRequest(newToken);
    } catch (error) {
      // Refresh failed, redirect to login
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
      throw error;
    } finally {
      isRefreshing = false;
    }
  }

  return response;
}

export function logoutClient() {
  clearCookie(ACCESS_TOKEN_COOKIE);
  clearCookie(REFRESH_TOKEN_COOKIE);
  if (typeof window !== "undefined") {
    window.location.href = "/login";
  }
}
