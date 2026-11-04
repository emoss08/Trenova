import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type LineProps = {
  /** The real line's type classes, which set the line's height. */
  className?: string;
  bar: string;
};

/**
 * A line of text waiting for its value. The zero-width space gives it the real line's line
 * box, so a row of these is exactly as tall as the row it stands in for.
 */
function Line({ className, bar }: LineProps) {
  return (
    <span className={cn("flex min-w-0 items-center", className)}>
      {"​"}
      <Skeleton className={cn("h-2.5 max-w-full", bar)} />
    </span>
  );
}

function FigureCell({ top, bottom }: { top: string; bottom: string }) {
  return (
    <div className="flex flex-col items-end gap-0.5 text-right">
      <Line className="font-table justify-end text-xs font-semibold" bar={top} />
      <Line className="font-table justify-end text-3xs" bar={bottom} />
    </div>
  );
}

/**
 * Each column's cell while its row loads, on the real cell's line boxes. The board's skeleton
 * and the table's own loading rows both draw these, so nothing moves between the two waits or
 * when the rows land. A line the real cell hides in compact density hides here too.
 */
export const BOARD_CELL_SKELETONS: Readonly<Record<string, ReactNode>> = {
  lane: (
    <div className="flex min-w-0 flex-col gap-0.5">
      <Line className="text-sm font-medium" bar="w-44" />
      <Line className="font-mono text-xs in-data-[density=compact]:hidden" bar="w-16" />
    </div>
  ),
  status: <Skeleton className="rounded-badge h-5 w-24" />,
  tenderStatus: <Line className="text-sm" bar="w-16" />,
  billing: <Skeleton className="rounded-badge h-5 w-20" />,
  proBol: (
    <div className="flex flex-col gap-0.5">
      <Line className="font-table font-medium" bar="w-24" />
      <Line className="font-table text-2xs" bar="w-20" />
    </div>
  ),
  order: <Line className="font-table" bar="w-20" />,
  customer: (
    <div className="flex min-w-0 flex-col gap-0.5">
      <Line className="text-sm" bar="w-36" />
      <Line className="font-mono text-xs in-data-[density=compact]:hidden" bar="w-24" />
    </div>
  ),
  driver: (
    <div className="flex min-w-0 items-center gap-2">
      <Skeleton className="size-6 shrink-0 rounded-full" />
      <div className="flex min-w-0 flex-col">
        <Line className="text-sm font-medium" bar="w-28" />
        <Line className="font-mono text-xs in-data-[density=compact]:hidden" bar="w-20" />
      </div>
    </div>
  ),
  eta: (
    <div className="flex min-w-0 flex-col">
      <Line className="font-mono text-sm" bar="w-28" />
      <Line className="font-mono text-xs in-data-[density=compact]:hidden" bar="w-14" />
    </div>
  ),
  revenue: <FigureCell top="w-16" bottom="w-12" />,
  margin: <FigureCell top="w-10" bottom="w-14" />,
  actions: (
    <div className="flex items-center justify-end gap-0.5">
      <Skeleton className="size-6 rounded-md" />
    </div>
  ),
};
