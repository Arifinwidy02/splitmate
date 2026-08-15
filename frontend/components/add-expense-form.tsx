"use client";

import { useState, useRef } from "react";
import { useActionState } from "react";
import Image from "next/image";
import { ImagePlus, X } from "lucide-react";

import { createExpense } from "@/app/actions/expenses";
import AmountInput from "@/components/amount-input";
import { CategoryIcon } from "@/components/category-icon";
import { EXPENSE_CATEGORIES, type Member, type User } from "@/lib/api";
import { toRFC3339 } from "@/lib/format";
import type { Dict } from "@/lib/i18n/id";
import { tr } from "@/lib/i18n/tr";

const MAX_RECEIPT_BYTES = 5 * 1024 * 1024;
const RECEIPT_TYPES = ["image/jpeg", "image/png", "image/webp", "image/gif"];

function localDateTimeValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(
    d.getHours(),
  )}:${pad(d.getMinutes())}`;
}

export default function AddExpenseForm({
  groupId,
  currency,
  members,
  user,
  dict,
}: {
  groupId: string;
  currency: string;
  members: Member[];
  user: User;
  dict: Dict;
}) {
  const [state, action, pending] = useActionState(createExpense.bind(null, groupId), undefined);
  const receiptInputRef = useRef<HTMLInputElement>(null);

  const [splitType, setSplitType] = useState<"equal" | "custom">("equal");
  const [selected, setSelected] = useState<Set<string>>(new Set([user.id]));
  const [customTotals, setCustomTotals] = useState<Record<string, string>>({});
  const [expenseDate, setExpenseDate] = useState<string>(() => localDateTimeValue(new Date()));
  const [category, setCategory] = useState<string>("Food & Drinks");
  const [receipt, setReceipt] = useState<File | null>(null);
  const [receiptPreview, setReceiptPreview] = useState<string | null>(null);
  const [receiptError, setReceiptError] = useState<string | null>(null);

  const toggle = (userId: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(userId)) {
        next.delete(userId);
      } else {
        next.add(userId);
      }
      return next;
    });
  };

  const onReceiptChange = (file: File | null) => {
    setReceiptError(null);

    if (!file) {
      setReceipt(null);
      setReceiptPreview(null);
      return;
    }

    if (!RECEIPT_TYPES.includes(file.type)) {
      setReceiptError(dict.expenseForm.receiptTypeError);
      setReceipt(null);
      setReceiptPreview(null);
      return;
    }
    if (file.size > MAX_RECEIPT_BYTES) {
      setReceiptError(dict.expenseForm.receiptSizeError);
      setReceipt(null);
      setReceiptPreview(null);
      return;
    }

    if (receiptPreview) URL.revokeObjectURL(receiptPreview);
    setReceipt(file);
    setReceiptPreview(URL.createObjectURL(file));
  };

  const removeReceipt = () => {
    if (receiptPreview) URL.revokeObjectURL(receiptPreview);
    setReceipt(null);
    setReceiptPreview(null);
    if (receiptInputRef.current) receiptInputRef.current.value = "";
  };

  return (
    <form action={action} className="flex flex-col gap-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <label htmlFor="description" className="text-sm font-medium text-slate-700">
            {dict.expenseForm.description}
          </label>
          <input
            id="description"
            name="description"
            type="text"
            required
            maxLength={255}
            placeholder={dict.expenseForm.descriptionPlaceholder}
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="amount" className="text-sm font-medium text-slate-700">
            {dict.expenseForm.amount}
          </label>
          <AmountInput
            id="amount"
            name="amount"
            required
            placeholder={dict.expenseForm.amountPlaceholder}
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>

        <fieldset>
          <legend className="text-sm font-medium text-slate-700">
            {dict.expenseForm.category}
          </legend>
          <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3">
            {EXPENSE_CATEGORIES.map((c) => (
              <label
                key={c}
                className={`flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-green-600/40 ${
                  category === c
                    ? "border-green-600 bg-green-50 font-medium text-green-800"
                    : "border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50"
                }`}
              >
                <input
                  type="radio"
                  name="category"
                  value={c}
                  checked={category === c}
                  onChange={() => setCategory(c)}
                  className="sr-only"
                />
                <CategoryIcon category={c} className="h-4 w-4" />
                {c}
              </label>
            ))}
          </div>
        </fieldset>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="paidBy" className="text-sm font-medium text-slate-700">
            {dict.expenseForm.paidBy}
          </label>
          <select
            id="paidBy"
            name="paidBy"
            defaultValue={user.id}
            className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          >
            {members.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="expenseDate" className="text-sm font-medium text-slate-700">
            {dict.expenseForm.date}
          </label>
          <input
            id="expenseDate"
            name="expenseDate"
            type="datetime-local"
            required
            value={expenseDate}
            onChange={(e) => setExpenseDate(e.target.value)}
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
          />
        </div>
      </div>

      <input type="hidden" name="expenseDateRfc" value={toRFC3339(expenseDate)} />
      <input type="hidden" name="currency" value={currency} />

      <fieldset>
        <legend className="text-sm font-medium text-slate-700">
          {dict.expenseForm.splitMethod}
        </legend>
        <div className="mt-2 grid grid-cols-2 gap-2 rounded-lg bg-slate-100 p-1">
          {(
            [
              ["equal", dict.expenseForm.splitEqually],
              ["custom", dict.expenseForm.customAmounts],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => setSplitType(value)}
              aria-pressed={splitType === value}
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition ${
                splitType === value
                  ? "bg-white text-slate-900 shadow-sm"
                  : "text-slate-500 hover:text-slate-700"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </fieldset>

      <input type="hidden" name="splitType" value={splitType} />

      <fieldset>
        <legend className="text-sm font-medium text-slate-700">
          {dict.expenseForm.splitBetween}
        </legend>
        <ul className="mt-2 flex flex-col">
          {members.map((m, i) => (
            <li
              key={m.id}
              className={i > 0 ? "border-t border-slate-100" : undefined}
            >
              <label
                htmlFor={`participant-${m.id}`}
                className="flex cursor-pointer items-center justify-between gap-3 py-2.5"
              >
                <span className="flex items-center gap-2 text-sm text-slate-800">
                  <input
                    id={`participant-${m.id}`}
                    type="checkbox"
                    name="participant"
                    value={m.id}
                    checked={selected.has(m.id)}
                    onChange={() => toggle(m.id)}
                    className="h-4 w-4 rounded border-slate-300 accent-green-600"
                  />
                  {m.name}
                  {m.id === user.id && (
                    <span className="text-xs text-slate-400">{dict.common.you}</span>
                  )}
                </span>
                {splitType === "custom" && selected.has(m.id) && (
                  <AmountInput
                    id={`split-${m.id}`}
                    name={`split-${m.id}`}
                    value={customTotals[m.id] ?? ""}
                    onChange={(parsed) =>
                      setCustomTotals((prev) => ({ ...prev, [m.id]: parsed }))
                    }
                    placeholder={dict.expenseForm.amountPlaceholder}
                    ariaLabel={tr(dict.expenseForm.shareFor, { name: m.name })}
                    className="w-28 rounded-lg border border-slate-200 px-2 py-1 text-right text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
                  />
                )}
              </label>
            </li>
          ))}
        </ul>
      </fieldset>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="note" className="text-sm font-medium text-slate-700">
          {dict.expenseForm.note}{" "}
          <span className="font-normal text-slate-400">{dict.common.optional}</span>
        </label>
        <input
          id="note"
          name="note"
          type="text"
          maxLength={1000}
          placeholder={dict.expenseForm.notePlaceholder}
          className="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-green-600 focus:ring-2 focus:ring-green-600/20"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-slate-700">
          {dict.expenseForm.receipt}{" "}
          <span className="font-normal text-slate-400">{dict.common.optional}</span>
        </span>

        {receiptPreview ? (
          <div className="flex items-center gap-3">
            <Image
              src={receiptPreview}
              alt={dict.expenseForm.receiptPreviewAlt}
              width={64}
              height={64}
              className="h-16 w-16 rounded-lg border border-slate-200 object-cover"
            />
            <div className="flex flex-col gap-1">
              <p className="max-w-[220px] truncate text-sm text-slate-700">{receipt?.name}</p>
              <button
                type="button"
                onClick={removeReceipt}
                className="inline-flex items-center gap-1 text-sm font-medium text-red-600 hover:underline"
              >
                <X className="h-3.5 w-3.5" aria-hidden="true" />
                {dict.expenseForm.removeReceipt}
              </button>
            </div>
          </div>
        ) : (
          <label
            htmlFor="receipt"
            className="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-slate-300 px-3 py-2.5 text-sm text-slate-500 transition hover:border-green-500 hover:text-green-700"
          >
            <ImagePlus className="h-4 w-4" aria-hidden="true" />
            {dict.expenseForm.receiptHint}
          </label>
        )}

        <input
          ref={receiptInputRef}
          id="receipt"
          name="receipt"
          type="file"
          accept="image/jpeg,image/png,image/webp,image/gif"
          onChange={(e) => onReceiptChange(e.target.files?.[0] ?? null)}
          className="sr-only"
        />

        {receiptError && (
          <p role="alert" className="text-sm text-red-600">
            {receiptError}
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
        {pending ? dict.expenseForm.adding : dict.expenseForm.addExpense}
      </button>
    </form>
  );
}
