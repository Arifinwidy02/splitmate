"use client";

import { useState } from "react";
import { Copy, Check, Share2, ExternalLink } from "lucide-react";

import { Card, CardHeader } from "@/components/card";
import QRCode from "@/components/qr-code";
import { getOrCreateInviteLink } from "@/app/actions/groups";
import type { Dict } from "@/lib/i18n/id";

interface InviteLinkCardProps {
  groupId: string;
  dict: Dict;
}

export default function InviteLinkCard({ groupId, dict }: InviteLinkCardProps) {
  const [link, setLink] = useState<string>("");
  const [isPending, setIsPending] = useState(false);
  const [copied, setCopied] = useState(false);

  const fetchLink = async () => {
    setIsPending(true);
    try {
      const result = await getOrCreateInviteLink(groupId);
      if (result && 'url' in result && result.url) {
        setLink(result.url);
      }
    } finally {
      setIsPending(false);
    }
  };

  const copyLink = async () => {
    if (!link) return;
    await navigator.clipboard.writeText(link);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const shareLink = async () => {
    if (!link) return;
    if (navigator.share) {
      try {
        await navigator.share({
          title: dict.groups.inviteLink,
          text: dict.groups.inviteLinkCreated,
          url: link,
        });
      } catch {
        // user cancelled
      }
    } else {
      copyLink();
    }
  };

  return (
    <Card>
      <CardHeader>
        <h3 className="text-xl font-semibold text-slate-900">{dict.groups.inviteLink}</h3>
      </CardHeader>
      <div className="p-5 space-y-4">
        {link ? (
          <>
            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium text-slate-700">
                {dict.groups.inviteLink}
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  readOnly
                  value={link}
                  className="flex-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm font-mono text-slate-800 outline-none"
                />
                <button
                  type="button"
                  onClick={copyLink}
                  aria-label={copied ? dict.groups.copied : dict.groups.copyLinkAria}
                  className="inline-flex items-center justify-center rounded-lg border border-slate-200 bg-white px-3 py-2 transition hover:bg-slate-50"
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-green-600" aria-hidden="true" />
                  ) : (
                    <Copy className="h-4 w-4" aria-hidden="true" />
                  )}
                </button>
                <button
                  type="button"
                  onClick={shareLink}
                  aria-label={dict.groups.share}
                  className="inline-flex items-center justify-center rounded-lg border border-slate-200 bg-white px-3 py-2 transition hover:bg-slate-50"
                >
                  <Share2 className="h-4 w-4" aria-hidden="true" />
                </button>
              </div>
            </div>

            <div className="flex items-center justify-center gap-4">
              <QRCode value={link} size={160} alt={dict.groups.qrCodeAlt} />
              <a
                href={link}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 text-sm font-medium text-green-600 hover:underline"
              >
                <ExternalLink className="h-4 w-4" aria-hidden="true" />
                {dict.groups.openLink}
              </a>
            </div>
          </>
        ) : (
          <div className="flex items-center justify-center gap-2">
            <button
              onClick={fetchLink}
              disabled={isPending}
              className="rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
            >
              {isPending ? dict.groups.inviting : dict.groups.createInviteLink}
            </button>
            <p className="text-sm text-slate-500">{dict.groups.tokenExpiry}</p>
          </div>
        )}
      </div>
    </Card>
  );
}