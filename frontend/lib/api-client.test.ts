import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { apiFetch } from "./api-client";

const originalLocation = window.location;

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("apiFetch", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: originalLocation,
    });
  });

  it("returns the response directly when the request succeeds", async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ data: { ok: true } }, 200));

    const res = await apiFetch("/api/v1/groups");

    expect(res.status).toBe(200);
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/groups",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("refreshes the access token on 401 and retries the request", async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ error: { code: "UNAUTHORIZED" } }, 401))
      .mockResolvedValueOnce(jsonResponse({ data: { user: { id: "1" } } }, 200))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }, 200));

    const res = await apiFetch("/api/v1/me");

    expect(res.status).toBe(200);
    expect(fetch).toHaveBeenCalledTimes(3);
    const refreshCall = vi.mocked(fetch).mock.calls.find(([url]) => url === "/api/v1/auth/refresh");
    expect(refreshCall).toBeDefined();
    expect(refreshCall?.[1]).toEqual(
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });

  it("throws when the refresh fails", async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ error: { code: "UNAUTHORIZED" } }, 401))
      .mockResolvedValueOnce(jsonResponse({ error: { code: "UNAUTHORIZED" } }, 401));

    await expect(apiFetch("/api/v1/groups")).rejects.toThrow("Failed to refresh token");
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("does not retry on 401 when skipAuth is set", async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ error: { code: "UNAUTHORIZED" } }, 401));

    const res = await apiFetch("/api/v1/groups", { skipAuth: true });

    expect(res.status).toBe(401);
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("refreshes only once when multiple requests fail concurrently", async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(jsonResponse({ error: { code: "UNAUTHORIZED" } }, 401))
      .mockResolvedValueOnce(jsonResponse({ error: { code: "UNAUTHORIZED" } }, 401))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }, 200))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }, 200))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }, 200));

    const [a, b] = await Promise.all([apiFetch("/api/v1/groups"), apiFetch("/api/v1/dashboard")]);

    expect(a.status).toBe(200);
    expect(b.status).toBe(200);
    const refreshCalls = vi
      .mocked(fetch)
      .mock.calls.filter(([url]) => url === "/api/v1/auth/refresh");
    expect(refreshCalls).toHaveLength(1);
  });
});