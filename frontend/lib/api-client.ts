interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
}

let isRefreshing = false;
let refreshFailed = false;
const refreshWaiters: (() => void)[] = [];

/**
 * Tokens are HttpOnly cookies, so they are never readable from
 * `document.cookie`. The refresh endpoint runs on the same origin (via the
 * Next.js rewrite proxy), so the browser sends the refresh token cookie
 * automatically and stores the new cookies from the response.
 */
async function refreshAccessToken(): Promise<boolean> {
  const res = await fetch("/api/v1/auth/refresh", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
  });
  return res.ok;
}

function redirectToLogin() {
  if (typeof window !== "undefined") {
    // Hard navigation is intentional: this runs outside React event handlers,
    // and a full reload ensures a fresh server-side session check after the
    // tokens have been cleared by the backend.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.href = "/login";
  }
}

export async function apiFetch(
  url: string,
  options: FetchOptions = {},
): Promise<Response> {
  const { skipAuth = false, ...fetchOptions } = options;

  const makeRequest = (): Promise<Response> => {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...(fetchOptions.headers as Record<string, string>),
    };

    return fetch(url, {
      ...fetchOptions,
      headers,
      credentials: "include",
    });
  };

  let response = await makeRequest();

  // If we get a 401, try to refresh the access token once and retry.
  if (response.status === 401 && !skipAuth) {
    if (isRefreshing) {
      // Another request is already refreshing; wait for it and retry.
      await new Promise<void>((resolve) => refreshWaiters.push(resolve));
      if (refreshFailed) {
        redirectToLogin();
        throw new Error("Failed to refresh token");
      }
      response = await makeRequest();
    } else {
      isRefreshing = true;
      refreshFailed = false;
      try {
        refreshFailed = !(await refreshAccessToken());
        if (refreshFailed) {
          throw new Error("Failed to refresh token");
        }
        refreshWaiters.splice(0).forEach((resolve) => resolve());
        response = await makeRequest();
      } catch (error) {
        refreshWaiters.splice(0).forEach((resolve) => resolve());
        redirectToLogin();
        throw error;
      } finally {
        isRefreshing = false;
      }
    }
  }

  return response;
}

export function logoutClient() {
  redirectToLogin();
}
