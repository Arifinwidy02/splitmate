"use client";

import Link from "next/link";
import { useActionState, useEffect, useState } from "react";
import { ArrowLeft, Users, Loader2 } from "lucide-react";

import { Card, CardHeader } from "@/components/card";
import { GroupLogo } from "@/components/group-logo";
import { formatDate } from "@/lib/format";
import type { GroupPreview } from "@/lib/api";
import { tr } from "@/lib/i18n/tr";
import { joinGroupViaLink } from "@/app/actions/groups";
import type { Dict } from "@/lib/i18n/id";

interface JoinPreviewClientProps {
  token: string;
  dict: Dict;
  initialPreview: GroupPreview | null;
  initialError: string | null;
}

export default function JoinPreviewClient({
  token,
  dict,
  initialPreview,
  initialError,
}: JoinPreviewClientProps) {
  const [preview, setPreview] = useState<GroupPreview | null>(initialPreview);
  const [loading, setLoading] = useState(!initialPreview && !initialError);
  const [error, setError] = useState<string | null>(initialError);
  const [isAuthed, setIsAuthed] = useState(false);
  const [joinState, joinAction, joinPending] = useActionState(joinGroupViaLink, undefined);

  const handleJoin = async () => {
    const fd = new FormData();
    fd.set("token", token);
    await joinAction(fd);
  };

  // Fetch preview without credentials (public endpoint)
  const fetchPreview = async () => {
    if (initialPreview || initialError) return;
    let mounted = true;
    try {
      const res = await fetch(`/api/invitations/${encodeURIComponent(token)}/preview`, {
        cache: "no-store",
      });
      if (!res.ok) throw new Error("Failed to fetch");
      const data = await res.json();
      if (mounted) setPreview(data.preview);
    } catch (err) {
      if (mounted) {
        const msg = err instanceof Error ? err.message : "Failed to fetch preview";
        setError(msg);
      }
    } finally {
      if (mounted) setLoading(false);
    }
    return () => { mounted = false; };
  };

  useEffect(() => {
    fetchPreview();
  }, [token, initialPreview, initialError]);

  // Check auth status
  useEffect(() => {
    fetch("/api/v1/auth/me", { credentials: "include", cache: "no-store" })
      .then(() => setIsAuthed(true))
      .catch(() => setIsAuthed(false));
  }, []);

  if (loading) {
    return (
      <div className="w-full max-w-md">
        <Card>
          <div className="py-12 text-center">
            <Loader2 className="mx-auto h-8 w-8 animate-spin text-green-600" aria-hidden="true" />
            <p className="mt-3 text-sm text-slate-500">{dict.common.loading}</p>
          </div>
        </Card>
      </div>
    );
  }

  if (error || !preview) {
    return (
      <div className="w-full max-w-md">
        <Card>
          <div className="py-12 text-center">
            <p className="text-sm text-red-700" role="alert">
              {error ?? dict.errors.somethingWentWrong}
            </p>
            <Link
              href="/groups"
              className="mt-4 inline-flex items-center gap-1.5 text-sm font-medium text-green-600 hover:underline"
            >
              <ArrowLeft className="h-4 w-4" aria-hidden="true" />
              {dict.groups.backToGroups}
            </Link>
          </div>
        </Card>
      </div>
    );
  }

  const joinUrl = `/login?next=/join/${encodeURIComponent(token)}`;
  const registerUrl = `/register?next=/join/${encodeURIComponent(token)}`;

  return (
    <div className="w-full max-w-md">
      <Card>
        <CardHeader className="text-center">
          <GroupLogo
            groupId={preview.id}
            hasLogo={preview.hasLogo}
            name={preview.name}
            className="mx-auto h-16 w-16 rounded-2xl bg-green-50 text-2xl font-bold text-green-700"
            imgClassName="h-16 w-16 rounded-2xl object-cover"
          />
          <h3 className="mt-4 text-xl font-semibold text-slate-900">{preview.name}</h3>
          {preview.description && (
            <p className="mt-1 text-sm text-slate-500">{preview.description}</p>
          )}
        </CardHeader>
        <div className="p-5 space-y-4">
          <div className="rounded-lg border border-slate-200 p-3">
            <dl className="grid grid-cols-2 gap-2 text-sm">
              <dt className="text-slate-500">{dict.groups.memberCountLabel}</dt>
              <dd className="font-medium text-slate-900 text-right">
                {tr(dict.groups.memberCount, { n: preview.memberCount })}
              </dd>
              <dt className="text-slate-500">{dict.groups.currency}</dt>
              <dd className="font-medium text-slate-900 text-right">{preview.currency}</dd>
              <dt className="text-slate-500">{dict.groups.createdAt}</dt>
              <dd className="font-medium text-slate-900 text-right">
                {formatDate(preview.createdAt)}
              </dd>
            </dl>
          </div>

          <div className="flex flex-col gap-3">
            {isAuthed ? (
              <button
                type="button"
                onClick={handleJoin}
                disabled={joinPending}
                className="w-full rounded-lg bg-green-600 px-4 py-3 text-lg font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
              >
                {joinPending ? dict.groups.joining : dict.groups.joinGroupBtn}
              </button>
            ) : (
              <>
                <a
                  href={joinUrl}
                  className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-3 text-lg font-semibold text-white transition hover:bg-green-700"
                >
                  {dict.groups.signInToJoin}
                </a>
                <a
                  href={registerUrl}
                  className="w-full inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 px-4 py-3 text-lg font-semibold text-slate-700 transition hover:bg-slate-50"
                >
                  {dict.groups.registerToJoin}
                </a>
              </>
            )}
          </div>

          {joinState?.error && (
            <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
              {joinState.error}
            </p>
          )}

          <p className="text-center text-xs text-slate-500">
            {dict.groups.previewSubtitle}
          </p>
        </div>
      </Card>
    </div>
  );
}