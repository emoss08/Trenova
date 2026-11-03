import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { AXIS_HEIGHT, LABEL_WIDTH } from "./timeline/timeline-scale";

/* Where each placeholder bar starts and how long it runs, as a share of the lane,
   so the loading sheet reads as a day of loads rather than a striped block. */
const BARS = [
  [0.08, 0.34],
  [0.22, 0.4],
  [0.3, 0.28],
  [0.12, 0.5],
  [0.46, 0.3],
  [0.38, 0.22],
  [0.55, 0.32],
  [0.18, 0.26],
] as const;

/**
 * The timeline's loading sheet: the toolbar, the axis, and rows with a label
 * and a bar each, at the sizes of what replaces them.
 */
export function ShipmentTimelineSkeleton() {
  const t = useT();

  return (
    <div
      className="flex h-full min-h-0 flex-col"
      aria-busy="true"
      aria-label={t("Loading the timeline")}
    >
      <div className="border-subtle flex h-10 shrink-0 items-center gap-3 border-b px-2.5">
        <Skeleton className="h-7 w-52 rounded-md" />
        <Skeleton className="h-7 w-56 rounded-md" />
        <Skeleton className="h-7 w-40 rounded-md" />
      </div>
      <ShipmentTimelineBodySkeleton />
    </div>
  );
}

/**
 * The sheet under the toolbar while a range loads: the toolbar stays live, so
 * stepping a day shows the new range loading rather than the old one emptied.
 */
export function ShipmentTimelineBodySkeleton() {
  return (
    <div className="flex min-h-0 flex-1 flex-col" aria-hidden>
      <div className="border-subtle flex shrink-0 border-b" style={{ height: AXIS_HEIGHT }}>
        <div className="border-subtle shrink-0 border-r" style={{ width: LABEL_WIDTH }} />
        <div className="flex flex-1 items-end gap-12 px-6 pb-2">
          {Array.from({ length: 8 }, (_, index) => (
            <Skeleton key={index} className="h-2.5 w-8" />
          ))}
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {BARS.map(([start, length], index) => (
          <div key={index} className="border-subtle flex h-10 border-b">
            <div
              className="border-subtle flex shrink-0 flex-col justify-center gap-1.5 border-r px-2.5"
              style={{ width: LABEL_WIDTH }}
            >
              <Skeleton className="h-3 w-32" />
              <Skeleton className="h-2.5 w-24" />
            </div>
            <div className="relative flex-1">
              <Skeleton
                className="absolute top-2.5 h-5 rounded-md"
                style={{ left: `${start * 100}%`, width: `${length * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
