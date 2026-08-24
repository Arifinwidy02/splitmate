import type { ComponentProps } from "react";

export function Card({ className = "", ...props }: ComponentProps<"section">) {
  return (
    <section
      className={`rounded-2xl border border-slate-200 bg-white p-5 shadow-card ${className}`}
      {...props}
    />
  );
}

export function CardHeader({ className = "", ...props }: ComponentProps<"h2">) {
  return (
    <h2
      className={`text-xl font-semibold tracking-tight text-slate-900 ${className}`}
      {...props}
    />
  );
}
