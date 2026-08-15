"use client";

import { useActionState, useRef, useState } from "react";
import Image from "next/image";
import { ImagePlus, X } from "lucide-react";

import { createGroup } from "@/app/actions/groups";
import type { Dict } from "@/lib/i18n/id";

const LOGO_TYPES = ["image/jpeg", "image/png", "image/webp", "image/gif"];
const MAX_LOGO_BYTES = 5 * 1024 * 1024;

export default function CreateGroupForm({ dict }: { dict: Dict }) {
  const [state, action, pending] = useActionState(createGroup, undefined);
  const logoInputRef = useRef<HTMLInputElement>(null);

  const [logo, setLogo] = useState<File | null>(null);
  const [logoPreview, setLogoPreview] = useState<string | null>(null);
  const [logoError, setLogoError] = useState<string | null>(null);

  const onLogoChange = (file: File | null) => {
    setLogoError(null);

    if (!file) {
      setLogo(null);
      setLogoPreview(null);
      return;
    }

    if (!LOGO_TYPES.includes(file.type)) {
      setLogoError(dict.groups.logoTypeError);
      setLogo(null);
      setLogoPreview(null);
      return;
    }
    if (file.size > MAX_LOGO_BYTES) {
      setLogoError(dict.groups.logoSizeError);
      setLogo(null);
      setLogoPreview(null);
      return;
    }

    if (logoPreview) URL.revokeObjectURL(logoPreview);
    setLogo(file);
    setLogoPreview(URL.createObjectURL(file));
  };

  const removeLogo = () => {
    if (logoPreview) URL.revokeObjectURL(logoPreview);
    setLogo(null);
    setLogoPreview(null);
    if (logoInputRef.current) logoInputRef.current.value = "";
  };

  return (
    <form action={action} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <label htmlFor="name" className="text-sm font-medium text-slate-700">
          {dict.groups.groupName}
        </label>
        <input
          id="name"
          name="name"
          type="text"
          required
          maxLength={100}
          placeholder={dict.groups.groupNamePlaceholder}
          className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="description" className="text-sm font-medium text-slate-700">
          {dict.groups.description}{" "}
          <span className="font-normal text-slate-400">{dict.common.optional}</span>
        </label>
        <input
          id="description"
          name="description"
          type="text"
          maxLength={500}
          placeholder={dict.groups.descriptionPlaceholder}
          className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="currency" className="text-sm font-medium text-slate-700">
          {dict.groups.currency}
        </label>
        <select
          id="currency"
          name="currency"
          defaultValue="IDR"
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        >
          <option value="IDR">{dict.groups.currencyIdr}</option>
          <option value="USD">{dict.groups.currencyUsd}</option>
          <option value="EUR">{dict.groups.currencyEur}</option>
          <option value="SGD">{dict.groups.currencySgd}</option>
        </select>
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-slate-700">
          {dict.groups.logo} <span className="font-normal text-slate-400">{dict.common.optional}</span>
        </span>

        {logoPreview ? (
          <div className="flex items-center gap-3">
            <Image
              src={logoPreview}
              alt={dict.groups.logoPreviewAlt}
              width={64}
              height={64}
              className="h-16 w-16 rounded-lg border border-slate-200 object-cover"
            />
            <div className="flex flex-col gap-1">
              <p className="max-w-[220px] truncate text-sm text-slate-700">{logo?.name}</p>
              <button
                type="button"
                onClick={removeLogo}
                className="inline-flex items-center gap-1 text-sm font-medium text-red-600 hover:underline"
              >
                <X className="h-3.5 w-3.5" aria-hidden="true" />
                {dict.groups.removeLogo}
              </button>
            </div>
          </div>
        ) : (
          <label
            htmlFor="logo"
            className="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-slate-300 px-3 py-2.5 text-sm text-slate-500 transition hover:border-green-500 hover:text-green-700"
          >
            <ImagePlus className="h-4 w-4" aria-hidden="true" />
            {dict.groups.logoHint}
          </label>
        )}

        <input
          ref={logoInputRef}
          id="logo"
          name="logo"
          type="file"
          accept="image/jpeg,image/png,image/webp,image/gif"
          onChange={(e) => onLogoChange(e.target.files?.[0] ?? null)}
          className="sr-only"
        />

        {logoError && (
          <p role="alert" className="text-sm text-red-600">
            {logoError}
          </p>
        )}
      </div>

      {state?.error && (
        <p
          role="alert"
          className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
        >
          {state.error}
        </p>
      )}

      <button
        type="submit"
        disabled={pending}
        className="rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-60"
      >
        {pending ? dict.groups.creating : dict.groups.create}
      </button>
    </form>
  );
}