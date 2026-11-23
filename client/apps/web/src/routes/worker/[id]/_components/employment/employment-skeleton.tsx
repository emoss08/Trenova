import type { WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine, ShimmerPanel } from "../tab-skeleton-parts";

/** History's own loading body: the year-grouped panel beside Dates and Leave standing. */
export function HistorySkeletonBody() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the employment history")}
      className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]"
    >
      <div aria-hidden className="flex flex-col overflow-hidden rounded-lg border bg-card">
        <div className="flex h-9 items-center justify-between gap-2 border-b px-3">
          <ShimmerLine className="w-36" />
          <ShimmerLine className="h-5 w-24 rounded-md" />
        </div>
        <div className="flex flex-col gap-4 p-3">
          {[0, 1, 2, 3, 4].map((row) => (
            <div key={row} className="grid grid-cols-[3.25rem_1rem_minmax(0,1fr)_5rem] gap-3">
              {row % 2 === 0 ? <ShimmerLine className="w-9" /> : <span />}
              <ShimmerLine className="size-3.5" />
              <div className="flex flex-col gap-1.5">
                <ShimmerLine className="w-32" />
                <ShimmerLine className="h-4 w-48" />
              </div>
              <ShimmerLine className="w-16" />
            </div>
          ))}
        </div>
      </div>
      <div aria-hidden className="flex flex-col gap-4">
        <ShimmerPanel rows={5} />
        <ShimmerPanel rows={2} />
      </div>
    </div>
  );
}

/** Checklists' own loading body: the open checklist beside Start and Closed. */
export function ChecklistsSkeletonBody() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the checklists")}
      className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]"
    >
      <div aria-hidden className="flex flex-col overflow-hidden rounded-lg border bg-card">
        <div className="flex h-9 items-center justify-between gap-2 border-b px-3">
          <ShimmerLine className="w-44" />
          <ShimmerLine className="h-7 w-28 rounded-md" />
        </div>
        <div className="flex items-end gap-4 border-b px-4 py-3">
          <ShimmerLine className="h-7 w-14" />
          <div className="flex flex-1 flex-col gap-2">
            <ShimmerLine className="h-2 w-full" />
            <ShimmerLine className="w-40" />
          </div>
        </div>
        <ShimmerPanel rows={6} className="rounded-none border-0" />
      </div>
      <div aria-hidden className="flex flex-col gap-4">
        <ShimmerPanel rows={1} />
        <ShimmerPanel rows={3} />
      </div>
    </div>
  );
}

/** The Employment tab's loading state, and its lazy fallback: the route band, then the view. */
export function EmploymentSkeleton({ view = "history" }: { view?: WorkerViewOf<"employment"> }) {
  return (
    <div className="flex flex-col gap-4">
      <LifecycleRouteSkeleton />
      {view === "history" ? <HistorySkeletonBody /> : <ChecklistsSkeletonBody />}
    </div>
  );
}

/** The route's loading state: the band with four stops in shimmer. */
export function LifecycleRouteSkeleton() {
  return (
    <div aria-hidden className="flex items-center gap-6 rounded-md border bg-card py-3 pr-3 pl-4">
      <div className="grid flex-1 grid-cols-4 gap-4">
        {[0, 1, 2, 3].map((stop) => (
          <div key={stop} className="flex flex-col gap-1.5">
            <span className="block size-2.5 ui-shimmer rounded-full" />
            <span className="block h-3 w-20 ui-shimmer rounded-sm" />
            <span className="block h-2.5 w-32 ui-shimmer rounded-sm" />
          </div>
        ))}
      </div>
      <span className="hidden h-7 w-28 ui-shimmer rounded-md lg:block" />
    </div>
  );
}
