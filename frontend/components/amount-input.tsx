"use client";

import { useState } from "react";

import { formatAmountInput, nextAmountInputValue } from "@/lib/amount";

export default function AmountInput({
  id,
  name,
  placeholder,
  className,
  required,
  value,
  onChange,
  ariaLabel,
}: {
  id: string;
  name: string;
  placeholder: string;
  className?: string;
  required?: boolean;
  value?: string;
  onChange?: (parsed: string) => void;
  ariaLabel?: string;
}) {
  const [internal, setInternal] = useState("");
  const parsed = value ?? internal;
  const display = formatAmountInput(parsed);

  return (
    <>
      <input
        id={id}
        type="text"
        inputMode="decimal"
        autoComplete="off"
        required={required}
        value={display}
        placeholder={placeholder}
        aria-label={ariaLabel}
        onChange={(e) => {
          const next = nextAmountInputValue(parsed, display, e.target.value);
          if (value === undefined) setInternal(next);
          onChange?.(next);
        }}
        className={className}
      />
      <input type="hidden" name={name} value={parsed} />
    </>
  );
}