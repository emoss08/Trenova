import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatDetentionMinutes,
  type DeskFloorStats,
  type DeskSummary,
} from "@trenova/shared/lib/detention";
import { pluralize } from "@trenova/shared/lib/utils";
import { KpiStrip } from "@/components/kpi/kpi-strip";
import { DeskMetric } from "./desk-metric";
import { DeskMoney } from "./desk-money";

type DeskSummaryRailProps = {
  summary: DeskSummary;
  floor: DeskFloorStats;
};

/**
 * What the floor is worth and how long it has been sitting. Four numbers in one
 * joined strip — the money reads as money because nothing around it is
 * competing for the same attention.
 */
export function DeskSummaryRail({ summary, floor }: DeskSummaryRailProps) {
  const t = useT();

  return (
    <KpiStrip>
      <DeskMetric
        label={t("Collectable now")}
        value={<DeskMoney value={summary.amountAtRisk} />}
        sub={
          summary.total === 0
            ? "Nothing on a dock"
            : `${summary.total} ${pluralize("stop", summary.total)} · avg ${formatDetentionMinutes(
                floor.averageOnSiteMinutes,
              )} on site`
        }
      />
      <DeskMetric
        label={t("Notice window")}
        value={summary.noticesDue}
        sub={
          summary.noticesDue > 0 ? "Must go out before the deadline" : "The notice queue is clear"
        }
      />
      <DeskMetric
        label={t("Uncollectable")}
        value={<DeskMoney value={summary.amountLost} precise={false} />}
        sub={`${summary.lost} ${pluralize("stop", summary.lost)} past the notice deadline`}
      />
      <DeskMetric
        label={t("Longest wait")}
        value={formatDetentionMinutes(floor.longestOnSiteMinutes)}
        sub={floor.longestLocationName || "Nothing on a dock"}
      />
    </KpiStrip>
  );
}
