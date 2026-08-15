"use client";

import { useState } from "react";

import { formatAmountInput, nextAmountInputValue } from "@/lib/amount";
import type { Locale } from "@/lib/i18n/constants";

export default function AmountInput({
  id,
  name,
  locale,
  placeholder,
  className,
  required,
}: {
  id: string;
  name: string;
  locale: Locale;
  placeholder: string;
  className?: string;
  required?: boolean;
}) {
  const [parsed, setParsed] = useState("");
  const display = formatAmountInput(parsed, locale);

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
        onChange={(e) =>
          setParsed((prev) =>
            nextAmountInputValue(prev, display, e.target.value),
          )
        }
        className={className}
      />
      <input type="hidden" name={name} value={parsed} />
    </>
  );
}