import { SkeletonRow } from "@/components/skeleton";

export default function Loading() {
  return (
    <div className="mx-auto w-full max-w-[1440px]">
      <div className="h-4 w-24 animate-pulse rounded bg-slate-200" />
      <div className="mt-4 flex items-center gap-4">
        <div className="h-12 w-12 animate-pulse rounded-2xl bg-slate-200" />
        <div>
          <div className="h-8 w-48 animate-pulse rounded bg-slate-200" />
          <div className="mt-2 h-4 w-64 animate-pulse rounded bg-slate-200" />
        </div>
      </div>
      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div className="flex flex-col gap-6 xl:col-span-2">
          <div className="animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card">
            <div className="h-6 w-32 rounded bg-slate-200" />
            <div className="mt-4">
              <SkeletonRow />
              <SkeletonRow />
              <SkeletonRow />
            </div>
          </div>
          <div className="animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card">
            <div className="h-6 w-36 rounded bg-slate-200" />
            <div className="mt-4">
              <SkeletonRow />
              <SkeletonRow />
            </div>
          </div>
        </div>
        <div className="flex flex-col gap-6">
          <div className="h-96 animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card" />
          <div className="h-48 animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-card" />
        </div>
      </div>
    </div>
  );
}
