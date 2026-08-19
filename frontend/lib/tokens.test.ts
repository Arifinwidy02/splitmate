import { describe, expect, it } from "vitest";

import { extractTokenFromSetCookie, isAccessTokenExpired } from "./tokens";

function makeJwt(payload: Record<string, unknown>): string {
  const enc = (obj: unknown) =>
    btoa(JSON.stringify(obj)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  return `${enc({ alg: "HS256", typ: "JWT" })}.${enc(payload)}.signature`;
}

describe("isAccessTokenExpired", () => {
  it("returns true when exp is in the past", () => {
    const token = makeJwt({ sub: "user-1", exp: Math.floor(Date.now() / 1000) - 60 });
    expect(isAccessTokenExpired(token)).toBe(true);
  });

  it("returns false when exp is in the future", () => {
    const token = makeJwt({ sub: "user-1", exp: Math.floor(Date.now() / 1000) + 3600 });
    expect(isAccessTokenExpired(token)).toBe(false);
  });

  it("returns true for a malformed token", () => {
    expect(isAccessTokenExpired("not-a-jwt")).toBe(true);
    expect(isAccessTokenExpired("a.b.c")).toBe(true);
  });

  it("returns true when exp is missing", () => {
    expect(isAccessTokenExpired(makeJwt({ sub: "user-1" }))).toBe(true);
  });
});

describe("extractTokenFromSetCookie", () => {
  it("extracts the value of a matching cookie", () => {
    expect(
      extractTokenFromSetCookie(
        "access_token=abc.def; Path=/; HttpOnly; Max-Age=900",
        "access_token",
      ),
    ).toBe("abc.def");
  });

  it("returns null when the cookie name does not match", () => {
    expect(
      extractTokenFromSetCookie("refresh_token=xyz; Path=/; HttpOnly", "access_token"),
    ).toBeNull();
  });

  it("returns null for an empty value", () => {
    expect(extractTokenFromSetCookie("access_token=; Path=/", "access_token")).toBeNull();
  });
});