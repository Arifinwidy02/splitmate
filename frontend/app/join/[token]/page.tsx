import { cookies } from "next/headers";

import { getDict } from "@/lib/i18n";
import { ACCESS_TOKEN_COOKIE } from "@/lib/tokens";
import JoinPreviewClient from "./join-preview-client";

interface JoinPageProps {
  params: Promise<{ token: string }>;
}

export default async function JoinPage({ params }: JoinPageProps) {
  const { token } = await params;
  const dict = await getDict();
  const cookieStore = await cookies();
  const initialIsAuthed = cookieStore.has(ACCESS_TOKEN_COOKIE);

  // Let the client component handle the preview fetch entirely
  // This avoids server-side fetch issues with localhost resolution
  return (
    <>
      <main className="flex flex-1 items-center justify-center bg-slate-50 p-8">
        <JoinPreviewClient
          token={token}
          dict={dict}
          initialPreview={null}
          initialError={null}
          initialIsAuthed={initialIsAuthed}
        />
      </main>
    </>
  );
}