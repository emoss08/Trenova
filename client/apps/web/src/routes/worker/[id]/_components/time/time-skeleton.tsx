import type { WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine, ShimmerPanel } from "../tab-skeleton-parts";

function RulersBand({ cells }: { cells: number }) {
  return (
    <div className="flex flex-col rounded-lg border">
      <div className="flex h-11 items-center gap-2 border-b px-4">
        <ShimmerLine className="h-5 w-20 rounded-sm" />
        <ShimmerLine className="w-56" />
      </div>
      <div className="grid grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: cells }, (_, index) => (
          <div key={index} className="flex flex-col gap-2 px-4 pt-3 pb-3.5">
            <ShimmerLine className="w-14" />
            <ShimmerLine className="h-5 w-24" />
            <ShimmerLine className="h-2.5 w-full rounded-xs" />
          </div>
        ))}
      </div>
    </div>
  );
}

function LogPanel() {
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border">
      <div className="flex h-9 items-center border-b px-3">
        <ShimmerLine className="w-24" />
      </div>
      <div className="flex flex-col gap-3 p-3">
        <ShimmerLine className="h-10 w-80 rounded-lg" />
        <ShimmerLine className="h-40 w-full rounded-md" />
      </div>
    </div>
  );
}

/** Each Time view's own layout in shimmer: the lazy fallback and the views' first read. */
export function TimeSkeleton({ view = "hours-of-service" }: { view?: WorkerViewOf<"time"> }) {
  const t = useT();
  const label =
    view === "hours-of-service"
      ? t("Loading the hours of service")
      : view === "schedule"
        ? t("Loading the schedule")
        : view === "time-off"
          ? t("Loading the time off")
          : t("Loading the leave");
  return (
    <div aria-busy aria-label={label} className="flex flex-col gap-4">
      <div aria-hidden className="contents">
        {view === "hours-of-service" ? (
          <>
            <RulersBand cells={4} />
            <LogPanel />
            <div className="grid items-start gap-4 lg:grid-cols-2">
              <ShimmerPanel rows={1} />
              <ShimmerPanel rows={2} />
            </div>
          </>
        ) : view === "schedule" ? (
          <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
            <div className="flex min-w-0 flex-col gap-4">
              <ShimmerPanel rows={1} />
              <ShimmerPanel rows={7} />
            </div>
            <ShimmerPanel rows={3} />
          </div>
        ) : view === "time-off" ? (
          <>
            <div className="flex items-center justify-between gap-3">
              <ShimmerLine className="w-64" />
              <ShimmerLine className="h-7 w-36 rounded-md" />
            </div>
            <RulersBand cells={3} />
            <div className="grid items-start gap-4 lg:grid-cols-2">
              <ShimmerPanel rows={4} />
              <ShimmerPanel rows={6} />
            </div>
          </>
        ) : (
          <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
            <ShimmerPanel rows={3} />
            <ShimmerPanel rows={4} />
          </div>
        )}
      </div>
    </div>
  );
}
