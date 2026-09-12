import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

  if (isLoading || !summary) {
    return <SummaryStripSkeleton />;
  }

  const toggleFocus = (bucket: UrgencyBucket) =>
    onUrgencyFocus(urgencyFocus === bucket ? null : bucket);

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
      <StatTile
        label={t("Needs Coverage")}
        hint={t("Moves in the window with no driver — click to show every urgency bucket.")}
        tone={summary.uncoveredMoves > 0 ? "warn" : undefined}
        clickable
        active={urgencyFocus === null}
        onClick={() => onUrgencyFocus(null)}
        value={<span className="tabular-nums">{summary.uncoveredMoves.toLocaleString()}</span>}
        sub={<span>{t("{0} already covered", summary.coveredMoves.toLocaleString())}</span>}
      />
      <StatTile
        label={t("Late")}
        hint={t("Pickup window already open, still uncovered — click to focus the Late lane.")}
        tone={summary.lateMoves > 0 ? "danger" : undefined}
        clickable={summary.lateMoves > 0}
        active={urgencyFocus === "Late"}
        onClick={summary.lateMoves > 0 ? () => toggleFocus("Late") : undefined}
        value={<span className="tabular-nums">{summary.lateMoves.toLocaleString()}</span>}
        sub={<span>{t("past the pickup window")}</span>}
      />
      <StatTile
        label={t("At Risk")}
        hint={t("Uncovered with a pickup inside 4 hours — click to focus the Next 4 Hours lane.")}
        tone={summary.atRiskMoves > 0 ? "warn" : undefined}
        clickable={summary.atRiskMoves > 0}
        active={urgencyFocus === "Now"}
        onClick={summary.atRiskMoves > 0 ? () => toggleFocus("Now") : undefined}
        value={<span className="tabular-nums">{summary.atRiskMoves.toLocaleString()}</span>}
        sub={<span>{t("pickup inside 4 hours")}</span>}
      />
      <StatTile
        label={t("Open Drivers")}
        hint={t("Available drivers holding no work — click to filter the capacity rail to them.")}
        tone={summary.unseatedDrivers > 0 ? "info" : undefined}
        clickable={summary.unseatedDrivers > 0}
        onClick={summary.unseatedDrivers > 0 ? () => onCapacityFilter("open") : undefined}
        value={<span className="tabular-nums">{summary.unseatedDrivers.toLocaleString()}</span>}
        sub={<span>{t("of {0} available", summary.availableDrivers.toLocaleString())}</span>}
      />
      <StatTile
        label={t("Utilization")}
        hint={t("Share of available drivers currently holding work; average empty miles across covered moves with a live position.")}
        value={<span className="tabular-nums">{Math.round(summary.utilizationPercent)}%</span>}
        sub={<span>{t("{0} avg deadhead", formatMiles(summary.averageDeadheadMiles))}</span>}
      />
      <StatTile
        label={t("Assigned Today")}
        hint={t("Assignments created since midnight.")}
        value={<span className="tabular-nums">{summary.assignedToday.toLocaleString()}</span>}
        sub={<span>{t("across the fleet")}</span>}
      />
    </div>
  );
}
