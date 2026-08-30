import { NextRequest, NextResponse } from "next/server";

import { API_URL } from "@/lib/api";

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ token: string }> }
) {
  const { token } = await params;

  try {
    const res = await fetch(`${API_URL}/api/v1/invitations/${encodeURIComponent(token)}/preview`, {
      cache: "no-store",
    });

    if (!res.ok) {
      const body = await res.json().catch(() => null);
      return NextResponse.json(
        { error: body?.error?.message ?? "Failed to fetch preview" },
        { status: res.status }
      );
    }

    const backend = await res.json();
    // backend shape: { data: { group: {...}, viewerIsMember } }
    const group = backend?.data?.group;
    if (!group) {
      return NextResponse.json({ error: "Invalid preview response" }, { status: 500 });
    }
    return NextResponse.json({ preview: group });
  } catch {
    return NextResponse.json(
      { error: "Failed to fetch preview" },
      { status: 500 }
    );
  }
}