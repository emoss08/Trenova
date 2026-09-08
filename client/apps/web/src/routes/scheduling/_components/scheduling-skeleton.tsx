import { KpiCard } from "@/components/kpi/kpi-card";
import { rotaCellMode, type RotaDensity } from "@/lib/scheduling-board";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { DAY_LABELS } from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: the page skeleton is the Suspense fallback while the
 * console's chunk arrives, and the board skeleton inside it is the rota tab's
 * own loading state while a week is read. Sharing the board between the two
 * keeps the page from reflowing between "loading the code" and "loading the
 * week".
 *
 * Every measurement mirrors the loaded layout: the KPI strip at the shared KPI
 * height, the tab strip, the week toolbar, the attention strip, a board with a
 * sticky worker column, seven day columns per week, a cover row and a week
 * total, and the legend under it. Widths are fixed so the shapes never
 * reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-20", "w-18", "w-14", "w-28"] as const;
const TAB_WIDTHS = ["w-14", "w-16", "w-16"] as const;
const ATTENTION_ROW_COUNT = 2;
const BOARD_ROW_COUNT = 8;
const WORKER_NAME_WIDTHS = [
  "w-24",
  "w-20",
  "w-28",
  "w-16",
  "w-24",
  "w-20",
  "w-28",
  "w-24",
] as const;
// Which days of the week each row works: a spread of patterns, so the board
// reads like a rota rather than a grid of identical boxes.
const ROW_MASKS = [
  0b0111110, 0b0111110, 0b1000011, 0b0111110, 0b1111100, 0b0111110, 0b0011111, 0b0111110,
] as const;
const CELL_HEIGHT = { detail: "h-12", time: "h-7", block: "h-6" } as const;
const COVER_BAR_WIDTHS = [
  "w-full",
  "w-full",
  "w-3/4",
  "w-full",
  "w-1/2",
  "w-1/4",
  "w-1/4",
] as const;
const LEGEND_COUNT = 6;

function OverviewSkeleton({ showSwaps }: { showSwaps: boolean }) {
  const count = showSwaps ? 4 : 3;
  return (
    <div className={cn("grid grid-cols-4 gap-3", showSwaps ? "lg:grid-cols-8" : "lg:grid-cols-6")}>
      {Array.from({ length: count }, (_, index) => (
        <KpiCard key={index} span={2}>
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", KPI_LABEL_WIDTHS[index])} />
          </div>
          {index === 1 ? (
            <div className="flex items-center gap-2.5">
              <Skeleton className="size-10 rounded-full" />
              <Skeleton className="h-6.5 w-14" />
            </div>
          ) : (
            <Skeleton className="h-6.5 w-10" />
          )}
          {index === 0 ? (
            <Skeleton className="mt-auto h-1.5 w-full rounded-full" />
          ) : (
            <Skeleton className="mt-auto h-2.5 w-3/4" />
          )}
        </KpiCard>
      ))}
    </div>
  );
}

function TabsSkeleton() {
  return (
    <div className="flex items-center gap-4 border-b">
      {TAB_WIDTHS.map((width, index) => (
        <span key={index} className="flex items-center gap-1.5 py-2">
          <Skeleton className="size-3.5 rounded-sm" />
          <Skeleton className={cn("h-3.5", width)} />
        </span>
      ))}
    </div>
  );
}

function ToolbarSkeleton() {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
        <div className="flex items-center gap-1">
          <Skeleton className="size-8 rounded-md" />
          <Skeleton className="mx-2 h-4 w-44" />
          <Skeleton className="size-8 rounded-md" />
          <Skeleton className="h-8 w-14 rounded-md" />
        </div>
        <Skeleton className="h-7 min-w-48 flex-1 rounded-md" />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Skeleton className="h-7 w-56 rounded-md" />
        <Skeleton className="h-7 w-52 rounded-md" />
        <Skeleton className="h-7 w-20 rounded-md" />
        <Skeleton className="h-8 w-24 rounded-md" />
      </div>
    </div>
  );
}

function AttentionSkeleton() {
  return (
    <section aria-label="Needs a look" className="bg-card overflow-hidden rounded-lg border">
      <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <Skeleton className="size-3.5 rounded-sm" />
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-4 w-5 rounded-full" />
        </div>
        <Skeleton className="h-6 w-28 rounded-md" />
      </header>
      <ul className="divide-y">
        {Array.from({ length: ATTENTION_ROW_COUNT }, (_, index) => (
          <li
            key={index}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
          >
            <span className="flex items-center gap-3">
              <Skeleton className="h-3.5 w-24" />
              <Skeleton className="h-3 w-40" />
            </span>
            <span className="flex items-center gap-2">
              <Skeleton className="h-4 w-16 rounded-full" />
              <Skeleton className="size-4 rounded-sm" />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

type RotaBoardSkeletonProps = {
  density?: RotaDensity;
  weeks?: number;
};

/**
 * The attention strip and the board, in the density and width the reader
 * chose, so the loaded week lands in the same place its outline was.
 */
export function RotaBoardSkeleton({ density = "comfortable", weeks = 1 }: RotaBoardSkeletonProps) {
  const compact = density === "compact";
  const mode = rotaCellMode(density, weeks);
  const cellHeight = mode === "block" && !compact ? "h-8" : CELL_HEIGHT[mode];
  const dayCount = DAY_LABELS.length * Math.max(1, weeks);
  const days = Array.from({ length: dayCount }, (_, index) => index);

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label="Loading the board">
      <div className="contents" aria-hidden>
        <AttentionSkeleton />
        <div
          data-density={density}
          data-cell-mode={mode}
          className="border-border max-h-[75vh] overflow-auto rounded-lg border"
        >
          <table
            aria-label="Rota"
            className={cn(
              "w-full border-separate border-spacing-0 text-xs",
              mode === "block" ? "min-w-[48rem]" : "min-w-[56rem]",
            )}
          >
            <thead>
              <tr>
                <th
                  className={cn(
                    "bg-muted sticky left-0 z-30 rounded-tl-lg px-3 text-left",
                    compact ? "w-40 py-1" : "w-56 py-2",
                  )}
                >
                  <Skeleton className="h-3 w-12" />
                </th>
                {days.map((day) => (
                  <th key={day} className={cn("bg-muted px-0.5", compact ? "py-1" : "py-2")}>
                    <span className="flex flex-col items-center gap-1">
                      <Skeleton className="h-2.5 w-6" />
                      {compact ? null : <Skeleton className="h-2.5 w-8" />}
                    </span>
                  </th>
                ))}
                <th className={cn("bg-muted rounded-tr-lg px-3", compact ? "py-1" : "py-2")}>
                  <Skeleton className="ml-auto h-3 w-10" />
                </th>
              </tr>
              <tr>
                <th className="bg-background sticky left-0 z-30 border-t border-b px-3 py-1 text-left">
                  <Skeleton className="h-2.5 w-10" />
                </th>
                {days.map((day) => (
                  <td
                    key={day}
                    className="bg-background border-t border-b px-0.5 py-1 align-bottom"
                  >
                    <span
                      className={cn("flex flex-col items-center", compact ? "gap-0.5" : "gap-1")}
                    >
                      <Skeleton className="h-2.5 w-4" />
                      <span className="bg-muted flex h-1 w-full max-w-16 overflow-hidden rounded-full">
                        <Skeleton
                          className={cn(
                            "h-full rounded-full",
                            COVER_BAR_WIDTHS[day % COVER_BAR_WIDTHS.length],
                          )}
                        />
                      </span>
                    </span>
                  </td>
                ))}
                <td className="bg-background border-t border-b px-3 py-1">
                  <Skeleton className="ml-auto h-2.5 w-10" />
                </td>
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: BOARD_ROW_COUNT }, (_, rowIndex) => {
                const mask = ROW_MASKS[rowIndex % ROW_MASKS.length] ?? 0;
                return (
                  <tr key={rowIndex}>
                    <td
                      className={cn(
                        "bg-background sticky left-0 z-10 border-t px-3 align-middle",
                        compact ? "py-0.5" : "py-1.5",
                      )}
                    >
                      {compact ? (
                        <Skeleton className={cn("h-3.5", WORKER_NAME_WIDTHS[rowIndex])} />
                      ) : (
                        <span className="flex items-center gap-2.5">
                          <Skeleton className="size-7 shrink-0 rounded-full" />
                          <span className="flex flex-col gap-1">
                            <Skeleton className={cn("h-3.5", WORKER_NAME_WIDTHS[rowIndex])} />
                            <Skeleton className="h-2.5 w-16" />
                          </span>
                        </span>
                      )}
                    </td>
                    {days.map((day) => {
                      const on = (mask >> (6 - (day % 7))) & 1;
                      return (
                        <td
                          key={day}
                          className={cn(
                            "border-t px-0.5 align-middle",
                            compact ? "py-0.5" : "py-1",
                          )}
                        >
                          <div
                            className={cn(
                              "flex w-full items-center justify-center rounded-md border",
                              cellHeight,
                              on ? "bg-muted/60" : "border-dashed",
                            )}
                          >
                            {on ? (
                              <Skeleton className={cn("h-2.5", mode === "block" ? "w-4" : "w-8")} />
                            ) : (
                              <Skeleton className="size-1.5 rounded-full" />
                            )}
                          </div>
                        </td>
                      );
                    })}
                    <td
                      className={cn(
                        "border-t px-3 text-right align-middle",
                        compact ? "py-0.5" : "py-1.5",
                      )}
                    >
                      <span className="flex flex-col items-end gap-1">
                        <Skeleton className="h-3 w-10" />
                        {compact ? null : <Skeleton className="h-2.5 w-12" />}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function LegendSkeleton() {
  return (
    <div className="flex flex-wrap items-center gap-4">
      {Array.from({ length: LEGEND_COUNT }, (_, index) => (
        <span key={index} className="flex items-center gap-1.5">
          <Skeleton className="size-2 rounded-full" />
          <Skeleton className="h-2.5 w-14" />
        </span>
      ))}
      <Skeleton className="ml-auto h-2.5 w-96 max-w-full" />
    </div>
  );
}

type SchedulingSkeletonProps = {
  /** Whether the strip carries the swaps card; false for somebody who may not read swaps. */
  showSwaps?: boolean;
};

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology. The sections keep the loaded
 * panels' labels only so the two trees can be compared like for like.
 */
export function SchedulingSkeleton({ showSwaps = true }: SchedulingSkeletonProps) {
  return (
    <div className="flex flex-col gap-4" aria-busy aria-label="Loading scheduling">
      <div className="contents" aria-hidden>
        <OverviewSkeleton showSwaps={showSwaps} />
        <TabsSkeleton />
        <ToolbarSkeleton />
        <RotaBoardSkeleton />
        <LegendSkeleton />
      </div>
    </div>
  );
}
