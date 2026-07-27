import type { DispatchBoardSummary } from "@/lib/graphql/dispatch-console";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

type Metric = {
  label: string;
  value: string;
  hint: string;
  tone?: "default" | "warning" | "danger";
};

/**
 * These are the numbers a dispatcher is measured on. Putting them above the board means
 * the cost of the decisions being made below is visible while they are being made.
 */
function buildMetrics(summary: DispatchBoardSummary): Metric[] {
  return [
    {
      label: "Uncovered",
      value: summary.uncoveredMoves.toLocaleString(),
      hint: "Moves in the window with no driver",
      tone: summary.uncoveredMoves > 0 ? "warning" : "default",
    },
    {
      label: "Late",
      value: summary.lateMoves.toLocaleString(),
      hint: "Pickup window already open, still uncovered",
      tone: summary.lateMoves > 0 ? "danger" : "default",
    },
    {
      label: "At risk",
      value: summary.atRiskMoves.toLocaleString(),
      hint: "Uncovered with a pickup inside 4 hours",
      tone: summary.atRiskMoves > 0 ? "warning" : "default",
    },
    {
      label: "Unseated",
      value: summary.unseatedDrivers.toLocaleString(),
      hint: `Of ${summary.availableDrivers.toLocaleString()} available drivers, holding no work`,
      tone: summary.unseatedDrivers > 0 ? "warning" : "default",
    },
    {
      label: "Utilization",
      value: `${Math.round(summary.utilizationPercent)}%`,
      hint: "Available drivers currently holding work",
    },
    {
      label: "Avg deadhead",
      value: `${Math.round(summary.averageDeadheadMiles).toLocaleString()} mi`,
      hint: "Empty miles on covered moves with a live position",
    },
    {
      label: "Assigned today",
      value: summary.assignedToday.toLocaleString(),
      hint: "Assignments created since midnight",
    },
  ];
}

const TONE_CLASS: Record<NonNullable<Metric["tone"]>, string> = {
  default: "text-foreground",
  warning: "text-amber-600 dark:text-amber-400",
  danger: "text-red-600 dark:text-red-400",
};

export function MetricsStrip({
  summary,
  isLoading,
}: {
  summary: DispatchBoardSummary | undefined;
  isLoading: boolean;
}) {
  if (isLoading || !summary) {
    return (
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7">
        {Array.from({ length: 7 }, (_, index) => (
          <Skeleton key={index} className="h-16 rounded-md" />
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7">
      {buildMetrics(summary).map((metric) => (
        <div
          key={metric.label}
          className="flex flex-col gap-0.5 rounded-md border border-border bg-card px-3 py-2"
          title={metric.hint}
        >
          <span className="text-[10px] uppercase tracking-wide text-muted-foreground">
            {metric.label}
          </span>
          <span
            className={cn(
              "text-lg font-semibold tabular-nums leading-none",
              TONE_CLASS[metric.tone ?? "default"],
            )}
          >
            {metric.value}
          </span>
        </div>
      ))}
    </div>
  );
}
