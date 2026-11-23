import type { WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine, ShimmerMainAside } from "../tab-skeleton-parts";

/** Each Files view's own layout in shimmer. */
export function FilesSkeleton({ view = "documents" }: { view?: WorkerViewOf<"files"> }) {
  const t = useT();
  const label =
    view === "documents"
      ? t("Loading the documents")
      : view === "driver-portal"
        ? t("Loading the driver portal")
        : t("Loading the agent memory");
  return (
    <div aria-busy aria-label={label} className="flex flex-col gap-4">
      <div aria-hidden className="contents">
        {view === "driver-portal" ? (
          <ShimmerMainAside main={3} aside={1} />
        ) : view === "documents" ? (
          <div className="flex flex-col overflow-hidden rounded-lg border">
            <div className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
              <ShimmerLine className="w-24" />
              <div className="flex gap-2">
                <ShimmerLine className="h-7 w-60 rounded-md" />
                <ShimmerLine className="h-7 w-32 rounded-md" />
                <ShimmerLine className="h-7 w-20 rounded-md" />
              </div>
            </div>
            <div className="grid grid-cols-[minmax(0,1fr)_8rem_8rem_6rem_4rem] gap-3 border-b px-3 py-2">
              {["w-12", "w-10", "w-14", "w-11", "ml-auto w-8"].map((width) => (
                <ShimmerLine key={width} className={width} />
              ))}
            </div>
            {Array.from({ length: 8 }, (_, row) => (
              <div
                key={row}
                className="grid grid-cols-[minmax(0,1fr)_8rem_8rem_6rem_4rem] items-center gap-3 border-b px-3 py-2.5 last:border-0"
              >
                <span className="flex items-center gap-2">
                  <span className="block size-6 ui-shimmer rounded-md" />
                  <ShimmerLine className="w-3/5" />
                </span>
                <ShimmerLine className="w-20" />
                <ShimmerLine className="w-24" />
                <ShimmerLine className="w-16" />
                <ShimmerLine className="ml-auto w-10" />
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-col overflow-hidden rounded-lg border">
            <div className="flex min-h-9 items-center justify-between border-b px-3 py-1.5">
              <ShimmerLine className="w-56" />
              <ShimmerLine className="w-12" />
            </div>
            {Array.from({ length: 4 }, (_, row) => (
              <div
                key={row}
                className="flex items-start gap-2.5 border-b px-3.5 py-2.5 last:border-0"
              >
                <span className="mt-0.5 block size-3.5 ui-shimmer rounded-xs" />
                <span className="flex flex-1 flex-col gap-1.5">
                  <ShimmerLine className="w-4/5" />
                  <ShimmerLine className="w-2/5" />
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
