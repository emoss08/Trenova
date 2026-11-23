import type { WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine, ShimmerPanel } from "../tab-skeleton-parts";

/** Training's own loading body: the standing row over the courses table. */
export function TrainingSkeletonBody() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the training")} className="flex flex-col gap-4">
      <div aria-hidden className="contents">
        <div className="flex flex-wrap items-center gap-5 rounded-lg border bg-card px-4 py-3">
          <ShimmerLine className="h-6 w-16" />
          <ShimmerLine className="w-28" />
          <ShimmerLine className="w-40" />
          <ShimmerLine className="h-5 w-20 rounded-sm" />
          <ShimmerLine className="ml-auto h-7 w-28 rounded-md" />
        </div>
        <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
          <div className="grid h-8 grid-cols-[minmax(0,2fr)_1fr_1fr_1fr_6rem] items-center gap-3 border-b bg-sunken px-3">
            {[0, 1, 2, 3, 4].map((head) => (
              <ShimmerLine key={head} className="w-16" />
            ))}
          </div>
          {[0, 1, 2, 3, 4, 5].map((row) => (
            <div
              key={row}
              className="grid grid-cols-[minmax(0,2fr)_1fr_1fr_1fr_6rem] items-center gap-3 border-b px-3 py-2.5 last:border-0"
            >
              <div className="flex flex-col gap-1.5">
                <ShimmerLine className="w-44" />
                <ShimmerLine className="h-2.5 w-28" />
              </div>
              <ShimmerLine className="w-20" />
              <ShimmerLine className="w-20" />
              <ShimmerLine className="w-20" />
              <ShimmerLine className="h-5 w-16 rounded-sm" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/** Reviews' own loading body: the open review beside Cadence and History. */
export function ReviewsSkeletonBody() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the reviews")}
      className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]"
    >
      <div aria-hidden className="flex flex-col overflow-hidden rounded-lg border bg-card">
        <div className="flex items-end justify-between gap-4 border-b px-4 pt-3 pb-3.5">
          <div className="flex flex-col gap-1.5">
            <ShimmerLine className="h-4 w-44" />
            <ShimmerLine className="w-64" />
          </div>
          <ShimmerLine className="h-7 w-20" />
        </div>
        {[0, 1, 2, 3, 4].map((row) => (
          <div
            key={row}
            className="grid grid-cols-[11rem_6rem_5rem_minmax(0,1fr)] items-center gap-4 border-b px-4 py-3 last:border-0"
          >
            <ShimmerLine className="w-32" />
            <ShimmerLine className="h-1.5 w-24" />
            <ShimmerLine className="w-14" />
            <ShimmerLine className="w-3/4" />
          </div>
        ))}
      </div>
      <div aria-hidden className="flex flex-col gap-4">
        <ShimmerPanel rows={2} />
        <ShimmerPanel rows={2} />
      </div>
    </div>
  );
}

/** The Development tab's loading state, and its lazy fallback. */
export function DevelopmentSkeleton({ view = "training" }: { view?: WorkerViewOf<"development"> }) {
  return view === "training" ? <TrainingSkeletonBody /> : <ReviewsSkeletonBody />;
}
