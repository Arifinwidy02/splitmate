"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Image from "next/image";
import { ChevronLeft, ChevronRight } from "lucide-react";

const SLIDES = [
  { src: "/traveling.jpg", alt: "Friends traveling together" },
  { src: "/eating-1.jpg", alt: "Friends sharing a meal" },
  { src: "/eating-2.jpg", alt: "Group dinner" },
  { src: "/eating-3.jpg", alt: "Sharing food together" },
] as const;

const AUTOPLAY_MS = 5000;

export function HeroCarousel() {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const touchStartX = useRef<number | null>(null);

  const go = useCallback((next: number) => {
    setIndex((next + SLIDES.length) % SLIDES.length);
  }, []);

  useEffect(() => {
    if (paused) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const t = setInterval(() => {
      setIndex((i) => (i + 1) % SLIDES.length);
    }, AUTOPLAY_MS);
    return () => clearInterval(t);
  }, [paused]);

  return (
    <div
      role="region"
      aria-roledescription="carousel"
      aria-label="Shared moments with SplitMate"
      className="absolute inset-0 touch-pan-y overflow-hidden"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={() => setPaused(false)}
      onTouchStart={(e) => {
        touchStartX.current = e.touches[0].clientX;
        setPaused(true);
      }}
      onTouchEnd={(e) => {
        if (touchStartX.current !== null) {
          const dx = e.changedTouches[0].clientX - touchStartX.current;
          if (Math.abs(dx) > 48) go(dx < 0 ? index + 1 : index - 1);
        }
        touchStartX.current = null;
        setPaused(false);
      }}
      onKeyDown={(e) => {
        if (e.key === "ArrowLeft") {
          e.preventDefault();
          go(index - 1);
        }
        if (e.key === "ArrowRight") {
          e.preventDefault();
          go(index + 1);
        }
      }}
    >
      {SLIDES.map((slide, i) => (
        <Image
          key={slide.src}
          src={slide.src}
          alt={slide.alt}
          fill
          priority={i === 0}
          sizes="(min-width: 1024px) 640px, 100vw"
          aria-hidden={i !== index}
          className={`object-cover transition-[opacity,transform] duration-700 ease-[cubic-bezier(0.16,1,0.3,1)] ${
            i === index
              ? "scale-100 opacity-100"
              : "pointer-events-none scale-[1.04] opacity-0"
          }`}
        />
      ))}

      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-black/35 to-transparent"
      />

      <button
        type="button"
        onClick={() => go(index - 1)}
        aria-label="Previous slide"
        className="absolute right-14 top-1/2 hidden h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-white/90 text-slate-700 shadow-sm transition hover:bg-white sm:flex"
      >
        <ChevronLeft className="h-5 w-5" aria-hidden="true" />
      </button>
      <button
        type="button"
        onClick={() => go(index + 1)}
        aria-label="Next slide"
        className="absolute right-4 top-1/2 hidden h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-white/90 text-slate-700 shadow-sm transition hover:bg-white sm:flex"
      >
        <ChevronRight className="h-5 w-5" aria-hidden="true" />
      </button>

      <div className="absolute bottom-4 right-4 flex items-center justify-center gap-2">
        {SLIDES.map((slide, i) => (
          <button
            key={slide.src}
            type="button"
            onClick={() => go(i)}
            aria-label={`Go to slide ${i + 1}`}
            aria-current={i === index}
            className={`h-2 rounded-full transition-all duration-300 ${
              i === index ? "w-6 bg-white" : "w-2 bg-white/60 hover:bg-white/90"
            }`}
          />
        ))}
      </div>

      <p aria-live="polite" className="sr-only">
        Slide {index + 1} of {SLIDES.length}
      </p>
    </div>
  );
}
