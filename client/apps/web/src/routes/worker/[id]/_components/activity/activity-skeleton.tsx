import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine } from "../tab-skeleton-parts";

/** The trail's own layout in shimmer: the filter row, a day heading and its rows. */
export function ActivitySkeleton() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the activity")} className="flex flex-col gap-4">
      <div aria-hidden className="flex flex-col overflow-hidden rounded-lg border">
        <div className="flex h-11 items-center justify-between gap-2 border-b px-3">
          <ShimmerLine className="w-20" />
          <div className="flex gap-2">
            <ShimmerLine className="h-7 w-56 rounded-md" />
            <ShimmerLine className="h-7 w-32 rounded-md" />
            <ShimmerLine className="h-7 w-32 rounded-md" />
          </div>
        </div>
        {[4, 3, 3].map((rows, day) => (
          <div key={day} className="flex flex-col">
            <div className="border-b px-3.5 pt-2.5 pb-1.5">
              <ShimmerLine className="w-28" />
            </div>
            {Array.from({ length: rows }, (_, row) => (
              <div
                key={row}
                className="grid grid-cols-[4.5rem_10.5rem_minmax(0,1fr)_3rem] items-center gap-3 px-3.5 py-2.5"
              >
                <ShimmerLine className="w-12" />
                <span className="flex items-center gap-2">
                  <span className="block size-6 ui-shimmer rounded-md" />
                  <ShimmerLine className="w-20" />
                </span>
                <ShimmerLine className="w-3/4" />
                <ShimmerLine className="w-10" />
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
