import { SkeletonCard, SkeletonRow } from "@/components/skeleton";

export default function Loading() {
  return (
    <div className="mx-auto w-full max-w-[1440px]">
      <div className="h-9 w-72 animate-pulse rounded bg-slate-200" />
      <div className="mt-3 h-4 w-56 animate-pulse rounded bg-slate-200" />

      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div className="animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card">
          <div className="h-6 w-40 rounded bg-slate-200" />
          <div className="mt-4">
            <SkeletonRow />
            <SkeletonRow />
            <SkeletonRow />
          </div>
        </div>
        <div className="animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card">
          <div className="h-6 w-44 rounded bg-slate-200" />
          <div className="mt-4">
            <SkeletonRow />
            <SkeletonRow />
            <SkeletonRow />
          </div>
        </div>
      </div>
    </div>
  );
}
