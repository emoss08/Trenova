import {
  formatDetentionMinutes,
  type DeskFloorStats,
  type DeskSummary,
} from "@trenova/shared/lib/detention";
import { pluralize } from "@trenova/shared/lib/utils";
import { DeskMetric } from "./desk-metric";
import { DeskMoney } from "./desk-money";

type DeskSummaryRailProps = {
  summary: DeskSummary;
  floor: DeskFloorStats;
};

/**
 * What the floor is worth and how long it has been sitting. Four numbers, no
 * boxes — the money reads as money because nothing around it is competing for
 * the same attention.
 */
export function DeskSummaryRail({ summary, floor }: DeskSummaryRailProps) {
  return (
    <dl className="sm:divide-border grid grid-cols-2 gap-x-6 gap-y-6 border-b p-4 pb-4 sm:grid-cols-4 sm:gap-y-0 sm:divide-x">
      <DeskMetric
        label="Collectable now"
        value={<DeskMoney value={summary.amountAtRisk} />}
        sub={
          summary.total === 0
            ? "Nothing on a dock"
            : `${summary.total} ${pluralize("stop", summary.total)} · avg ${formatDetentionMinutes(
                floor.averageOnSiteMinutes,
              )} on site`
        }
        className="sm:pr-6"
      />
      <DeskMetric
        label="Notice window"
        value={summary.noticesDue}
        sub={
          summary.noticesDue > 0 ? "Must go out before the deadline" : "The notice queue is clear"
        }
        className="sm:px-6"
      />
      <DeskMetric
        label="Uncollectable"
        value={<DeskMoney value={summary.amountLost} precise={false} />}
        sub={`${summary.lost} ${pluralize("stop", summary.lost)} past the notice deadline`}
        className="sm:px-6"
      />
      <DeskMetric
        label="Longest wait"
        value={formatDetentionMinutes(floor.longestOnSiteMinutes)}
        sub={floor.longestLocationName || "Nothing on a dock"}
        className="sm:pl-6"
      />
    </dl>
  );
}
