import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { apiFetchClient } from "./api-client";

const originalLocation = window.location;

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("apiFetchClient", () => {
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

    const res = await apiFetchClient<{ ok: boolean }>("/api/v1/groups");

    expect(res.ok).toBe(true);
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/groups",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("throws on non-ok response", async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ error: { code: "NOT_FOUND" } }, 404));

    await expect(apiFetchClient("/api/v1/groups/1")).rejects.toThrow();
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("includes credentials in requests", async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ data: { ok: true } }, 200));

    await apiFetchClient("/api/v1/me");

    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ credentials: "include" }),
    );
  });
});