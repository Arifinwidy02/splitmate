import { NextRequest, NextResponse } from "next/server";

import { API_URL } from "@/lib/api";

export async function GET(request: NextRequest) {
  try {
    const cookieHeader = request.headers.get("cookie") ?? "";

    const res = await fetch(`${API_URL}/api/v1/auth/me`, {
      headers: {
        Cookie: cookieHeader,
      },
      cache: "no-store",
    });

    if (!res.ok) {
      return NextResponse.json(
        { error: { code: "UNAUTHORIZED", message: "Authentication required" } },
        { status: 401 },
      );
    }

    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json(
      { error: { code: "INTERNAL", message: "Failed to fetch user" } },
      { status: 500 },
    );
  }
}
