export function SkeletonCard({ className = "" }: { className?: string }) {
  return (
    <div
      aria-hidden="true"
      className={`animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card ${className}`}
    >
      <div className="h-4 w-24 rounded bg-slate-200" />
      <div className="mt-3 h-8 w-36 rounded bg-slate-200" />
    </div>
  );
}

export function SkeletonRow({ className = "" }: { className?: string }) {
  return (
    <div
      aria-hidden="true"
      className={`animate-pulse py-3 ${className}`}
    >
      <div className="flex items-center gap-3">
        <div className="h-10 w-10 shrink-0 rounded-xl bg-slate-200" />
        <div className="flex-1">
          <div className="h-4 w-2/5 rounded bg-slate-200" />
          <div className="mt-1.5 h-3 w-1/3 rounded bg-slate-200" />
        </div>
        <div className="h-4 w-16 rounded bg-slate-200" />
      </div>
    </div>
  );
}
