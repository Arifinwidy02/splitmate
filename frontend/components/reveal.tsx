"use client";

import { useEffect, useRef, useState } from "react";

type RevealProps = {
  children: React.ReactNode;
  className?: string;
  delay?: number;
};

export default function Reveal({ children, className = "", delay = 0 }: RevealProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<"init" | "hidden" | "shown">("init");

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    setState("hidden");

    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setState("shown");
          io.disconnect();
        }
      },
      { threshold: 0.12 },
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);

  const hidden = state === "hidden";
  const shown = state === "shown";

  return (
    <div
      ref={ref}
      className={`${className} ${
        shown
          ? "translate-y-0 opacity-100 transition-[opacity,transform] duration-700 ease-[cubic-bezier(0.16,1,0.3,1)]"
          : hidden
            ? "translate-y-5 opacity-0"
            : ""
      }`}
      style={shown && delay ? { transitionDelay: `${delay}ms` } : undefined}
    >
      {children}
    </div>
  );
}
