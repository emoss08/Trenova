import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

const CHART_BARS = [42, 58, 50, 66, 61, 74, 69, 80, 72, 86, 78, 90] as const;

/** Every chart body on the dashboard is this tall, so panels sharing a row line up. */
export const PANEL_CHART_HEIGHT = "h-56";

/** Every list body on the dashboard is this tall, and scrolls past it. */
export const PANEL_LIST_HEIGHT = "h-80";

/**
 * A chart panel's body while its series loads: an axis gutter and bars at the
 * chart's own height, so the plot lands where the placeholder stood.
 */
export function ChartBodySkeleton({ label }: { label: string }) {
  return (
    <div className={cn("flex gap-3 p-3", PANEL_CHART_HEIGHT)} aria-busy aria-label={label}>
      <div className="flex w-10 shrink-0 flex-col justify-between py-1" aria-hidden>
        {Array.from({ length: 4 }, (_, index) => (
          <Skeleton key={index} className="h-2 w-8" />
        ))}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-2" aria-hidden>
        <div className="flex flex-1 items-end gap-1.5">
          {CHART_BARS.map((height, index) => (
            <Skeleton
              key={index}
              className="flex-1 rounded-b-none"
              style={{ height: `${height}%` }}
            />
          ))}
        </div>
        <div className="flex justify-between">
          {Array.from({ length: 4 }, (_, index) => (
            <Skeleton key={index} className="h-2 w-10" />
          ))}
        </div>
      </div>
    </div>
  );
}

/**
 * A list panel's body while its rows load: each row at the height and with
 * the columns the real row draws.
 */
export function RowsBodySkeleton({
  label,
  rows = 6,
  lead = "badge",
}: {
  label: string;
  rows?: number;
  /** What opens each row: a severity badge, or a rank number. */
  lead?: "badge" | "rank";
}) {
  return (
    <div
      className={cn("divide-border-subtle divide-y overflow-hidden", PANEL_LIST_HEIGHT)}
      aria-busy
      aria-label={label}
    >
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="flex items-center gap-3 px-3 py-2.5" aria-hidden>
          <Skeleton className={cn("shrink-0", lead === "badge" ? "h-5 w-16" : "h-3 w-4")} />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <div className="flex justify-between gap-3">
              <Skeleton className="h-2.5 w-2/5" />
              <Skeleton className="h-2.5 w-16" />
            </div>
            <Skeleton className="h-2 w-1/4" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** A chart panel's padded body, at the height its skeleton holds. */
export function PanelChartBody({ children }: { children: ReactNode }) {
  return <div className={cn("p-3", PANEL_CHART_HEIGHT)}>{children}</div>;
}

/** The quiet line a panel draws when it has nothing to show, at the body's own height. */
export function PanelQuiet({
  children,
  height = PANEL_CHART_HEIGHT,
  icon,
}: {
  children: ReactNode;
  height?: string;
  icon?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "text-muted-foreground flex flex-col items-center justify-center gap-2 px-3 text-center text-sm",
        height,
      )}
    >
      {icon}
      {children}
    </div>
  );
}
