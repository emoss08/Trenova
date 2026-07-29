import { StatTile } from "@/components/stat-tile";
import type { DispatchBoardSummary } from "@/lib/graphql/dispatch-console";
import { SummaryStripSkeleton } from "./console-skeletons";
import { formatMiles } from "./dispatch-vocabulary";
import type { CapacityFilter, UrgencyBucket } from "./dispatch-vocabulary";

export function SummaryStrip({
  summary,
  isLoading,
  urgencyFocus,
  onUrgencyFocus,
  onCapacityFilter,
}: {
  summary: DispatchBoardSummary | undefined;
  isLoading: boolean;
  urgencyFocus: UrgencyBucket | null;
  onUrgencyFocus: (bucket: UrgencyBucket | null) => void;
  onCapacityFilter: (filter: CapacityFilter) => void;
}) {
  if (isLoading || !summary) {
    return <SummaryStripSkeleton />;
  }

  const toggleFocus = (bucket: UrgencyBucket) =>
    onUrgencyFocus(urgencyFocus === bucket ? null : bucket);

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
      <StatTile
        label="Needs Coverage"
        hint="Moves in the window with no driver — click to show every urgency bucket."
        tone={summary.uncoveredMoves > 0 ? "warn" : undefined}
        clickable
        active={urgencyFocus === null}
        onClick={() => onUrgencyFocus(null)}
        value={<span className="tabular-nums">{summary.uncoveredMoves.toLocaleString()}</span>}
        sub={<span>{summary.coveredMoves.toLocaleString()} already covered</span>}
      />
      <StatTile
        label="Late"
        hint="Pickup window already open, still uncovered — click to focus the Late lane."
        tone={summary.lateMoves > 0 ? "danger" : undefined}
        clickable={summary.lateMoves > 0}
        active={urgencyFocus === "Late"}
        onClick={summary.lateMoves > 0 ? () => toggleFocus("Late") : undefined}
        value={<span className="tabular-nums">{summary.lateMoves.toLocaleString()}</span>}
        sub={<span>past the pickup window</span>}
      />
      <StatTile
        label="At Risk"
        hint="Uncovered with a pickup inside 4 hours — click to focus the Next 4 Hours lane."
        tone={summary.atRiskMoves > 0 ? "warn" : undefined}
        clickable={summary.atRiskMoves > 0}
        active={urgencyFocus === "Now"}
        onClick={summary.atRiskMoves > 0 ? () => toggleFocus("Now") : undefined}
        value={<span className="tabular-nums">{summary.atRiskMoves.toLocaleString()}</span>}
        sub={<span>pickup inside 4 hours</span>}
      />
      <StatTile
        label="Open Drivers"
        hint="Available drivers holding no work — click to filter the capacity rail to them."
        tone={summary.unseatedDrivers > 0 ? "info" : undefined}
        clickable={summary.unseatedDrivers > 0}
        onClick={summary.unseatedDrivers > 0 ? () => onCapacityFilter("open") : undefined}
        value={<span className="tabular-nums">{summary.unseatedDrivers.toLocaleString()}</span>}
        sub={<span>of {summary.availableDrivers.toLocaleString()} available</span>}
      />
      <StatTile
        label="Utilization"
        hint="Share of available drivers currently holding work; average empty miles across covered moves with a live position."
        value={<span className="tabular-nums">{Math.round(summary.utilizationPercent)}%</span>}
        sub={<span>{formatMiles(summary.averageDeadheadMiles)} avg deadhead</span>}
      />
      <StatTile
        label="Assigned Today"
        hint="Assignments created since midnight."
        value={<span className="tabular-nums">{summary.assignedToday.toLocaleString()}</span>}
        sub={<span>across the fleet</span>}
      />
    </div>
  );
}
