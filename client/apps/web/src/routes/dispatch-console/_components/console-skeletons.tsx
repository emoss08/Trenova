import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { CAPACITY_FILTERS, URGENCY_ORDER } from "./dispatch-vocabulary";
import {
  BAR_HEIGHT_PX,
  RAIL_WIDTH_PX,
  ROW_PADDING_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  TIMELINE_ROW_HEIGHT_PX,
} from "./timeline-metrics";

/**
 * Every pane of the console is code-split, so these stand in twice: as the Suspense
 * fallback while a pane's chunk arrives, and as the pane's own loading state while the
 * board query runs. Sharing them is what keeps the two indistinguishable — the layout
 * never shifts between "loading the code" and "loading the data".
 */

const RAIL_CHIP_WIDTHS = ["w-9", "w-14", "w-18", "w-16", "w-16", "w-18"] as const;
const CAPACITY_ROW_COUNT = 8;
const KANBAN_CARD_COUNTS = [4, 3, 3, 2, 2] as const;
const TIMELINE_ROW_COUNT = 9;
// Deterministic so the bars do not reshuffle between renders of the same skeleton.
const TIMELINE_BARS: readonly { left: string; width: string }[][] = [
  [
    { left: "4%", width: "18%" },
    { left: "34%", width: "26%" },
  ],
  [{ left: "12%", width: "30%" }],
  [
    { left: "6%", width: "14%" },
    { left: "48%", width: "22%" },
  ],
  [{ left: "22%", width: "38%" }],
  [
    { left: "2%", width: "22%" },
    { left: "56%", width: "18%" },
  ],
];

export function SummaryStripSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
      {Array.from({ length: 6 }, (_, index) => (
        <Skeleton key={index} className="h-20 rounded-lg" />
      ))}
    </div>
  );
}

export function CapacityRailRowsSkeleton() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: CAPACITY_ROW_COUNT }, (_, index) => (
        <div
          key={index}
          className="grid grid-cols-[12px_minmax(0,1fr)] gap-x-1.5 border-b px-2 py-2 last:border-b-0"
        >
          <span />
          <div className="flex min-w-0 flex-col gap-1.5">
            <div className="flex items-center gap-2">
              <Skeleton className="size-6 shrink-0 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <div className="flex items-center justify-between gap-2">
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-2.5 w-10" />
                </div>
                <Skeleton className="h-2.5 w-36" />
              </div>
            </div>
            <div className="flex items-center gap-2">
              {Array.from({ length: 3 }, (_, clock) => (
                <div key={clock} className="flex flex-1 items-center gap-1.5">
                  <Skeleton className="size-6 shrink-0 rounded-full" />
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <Skeleton className="h-2 w-8" />
                    <Skeleton className="h-2.5 w-9" />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

export function CapacityRailSkeleton() {
  return (
    <section className="bg-card flex min-h-0 flex-col overflow-hidden rounded-lg border">
      <header className="flex flex-col gap-2 border-b p-2">
        <Skeleton className="h-8 w-full rounded-md" />
        <div className="flex flex-wrap gap-1">
          {CAPACITY_FILTERS.map((filter, index) => (
            <Skeleton
              key={filter.id}
              className={`h-5 rounded-full ${RAIL_CHIP_WIDTHS[index] ?? "w-14"}`}
            />
          ))}
        </div>
      </header>

      <div className="flex items-center justify-between border-b px-2.5 py-1.5">
        <Skeleton className="h-2.5 w-16" />
        <Skeleton className="h-2.5 w-12" />
      </div>

      <div className="min-h-0 flex-1 overflow-hidden">
        <CapacityRailRowsSkeleton />
      </div>
    </section>
  );
}

export function CoverageKanbanSkeleton() {
  return (
    <div className="flex h-full min-h-0 gap-2 overflow-hidden">
      {URGENCY_ORDER.map((bucket, index) => (
        <div
          key={bucket}
          className="bg-muted/30 flex min-h-0 w-60 shrink-0 flex-col rounded-md border xl:w-auto xl:flex-1"
        >
          <header className="flex items-center gap-1.5 border-b px-2 py-1.5">
            <Skeleton className="size-1.5 rounded-full" />
            <Skeleton className="h-2.5 w-20" />
            <Skeleton className="ml-auto h-2.5 w-4" />
          </header>
          <div className="flex min-h-0 flex-col gap-1.5 overflow-hidden p-1.5">
            {Array.from({ length: KANBAN_CARD_COUNTS[index] ?? 2 }, (_, card) => (
              <Skeleton key={card} className="h-27 shrink-0 rounded-md" />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function DispatchTimelineSkeleton() {
  return (
    <div className="relative h-full overflow-hidden">
      <div
        className="border-border bg-muted flex border-b"
        style={{ height: TIMELINE_HEADER_HEIGHT_PX }}
      >
        <div
          className="border-border flex shrink-0 items-center border-r px-2.5"
          style={{ width: RAIL_WIDTH_PX }}
        >
          <Skeleton className="h-2.5 w-20" />
        </div>
        <div className="flex flex-1 items-center gap-10 px-2">
          {Array.from({ length: 6 }, (_, index) => (
            <Skeleton key={index} className="h-2.5 w-14" />
          ))}
        </div>
      </div>

      {Array.from({ length: TIMELINE_ROW_COUNT }, (_, index) => (
        <div
          key={index}
          className="border-border/70 flex border-b"
          style={{ height: TIMELINE_ROW_HEIGHT_PX }}
        >
          <div
            className="border-border flex shrink-0 items-center gap-1.5 border-r px-2"
            style={{ width: RAIL_WIDTH_PX }}
          >
            <Skeleton className="size-6 shrink-0 rounded-full" />
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <Skeleton className="h-2.5 w-24" />
              <Skeleton className="h-2 w-16" />
            </div>
          </div>
          <div className="relative flex-1">
            {(TIMELINE_BARS[index % TIMELINE_BARS.length] ?? []).map((bar) => (
              <Skeleton
                key={bar.left}
                className="absolute rounded"
                style={{
                  left: bar.left,
                  width: bar.width,
                  height: BAR_HEIGHT_PX,
                  top:
                    ROW_PADDING_PX / 2 +
                    (TIMELINE_ROW_HEIGHT_PX - ROW_PADDING_PX - BAR_HEIGHT_PX) / 2,
                }}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function InspectorSkeleton() {
  return (
    <section className="bg-card flex min-h-0 flex-col overflow-hidden rounded-lg border">
      <header className="flex items-center justify-between border-b px-2.5 py-1.5">
        <Skeleton className="h-2.5 w-20" />
      </header>

      <div className="flex flex-col gap-1 border-b px-2.5 py-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-2.5 w-40" />
        <Skeleton className="h-2.5 w-32" />
      </div>

      <div className="flex items-center justify-between gap-2 border-b px-2.5 py-1.5">
        <Skeleton className="h-2.5 w-20" />
        <Skeleton className="h-6 w-24 rounded-md" />
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-1.5 overflow-hidden p-2">
        {Array.from({ length: 5 }, (_, index) => (
          <Skeleton key={index} className="h-24 shrink-0 rounded-md" />
        ))}
      </div>
    </section>
  );
}
