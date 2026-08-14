import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, apiFetch } from "./server-api";

vi.mock("next/headers", () => ({
  cookies: async () => ({ toString: () => "session=test-token" }),
}));

describe("apiFetch", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("forwards the session cookie and returns the unwrapped data", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ data: { id: "abc" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const data = await apiFetch<{ id: string }>("/api/v1/groups");

    expect(data).toEqual({ id: "abc" });
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect((init?.headers as Record<string, string>).Cookie).toBe("session=test-token");
  });

  it("throws ApiError with code and message on error responses", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ error: { code: "GROUP_NOT_FOUND", message: "Not found" } }), {
        status: 404,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(apiFetch("/api/v1/groups/x")).rejects.toMatchObject({
      code: "GROUP_NOT_FOUND",
      message: "Not found",
    });
  });

  it("throws ApiError with INTERNAL when the error body is not JSON", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("boom", { status: 500 }));

    await expect(apiFetch("/api/v1/groups")).rejects.toBeInstanceOf(ApiError);
    await expect(apiFetch("/api/v1/groups")).rejects.toMatchObject({
      code: "INTERNAL",
    });
  });
});
