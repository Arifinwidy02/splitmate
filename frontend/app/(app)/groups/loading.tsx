import { SkeletonCard, SkeletonRow } from "@/components/skeleton";

export default function Loading() {
  return (
    <div className="mx-auto w-full max-w-[1440px]">
      <div className="h-9 w-52 animate-pulse rounded bg-slate-200" />
      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div className="animate-pulse rounded-2xl border border-slate-200 bg-white p-5 shadow-sm xl:col-span-2">
          <div className="h-6 w-36 rounded bg-slate-200" />
          <div className="mt-4">
            <SkeletonRow />
            <SkeletonRow />
            <SkeletonRow />
          </div>
        </div>
        <div className="flex flex-col gap-6">
          <SkeletonCard className="h-64" />
          <SkeletonCard className="h-40" />
        </div>
      </div>
    </div>
  );
}
