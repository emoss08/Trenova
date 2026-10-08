import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatDetentionMinutes,
  type DeskFloorStats,
  type DeskSummary,
} from "@trenova/shared/lib/detention";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
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
      <KpiStripItem
        label={t("Collectable now")}
        value={<DeskMoney value={summary.amountAtRisk} />}
        sub={
          summary.total === 0
            ? t("Nothing on a dock")
            : t(
                "{0, plural, one {# stop} other {# stops}} · avg {1} on site",
                summary.total,
                formatDetentionMinutes(floor.averageOnSiteMinutes),
              )
        }
      />
      <KpiStripItem
        label={t("Notice window")}
        value={summary.noticesDue}
        sub={
          summary.noticesDue > 0
            ? t("Must go out before the deadline")
            : t("The notice queue is clear")
        }
      />
      <KpiStripItem
        label={t("Uncollectable")}
        value={<DeskMoney value={summary.amountLost} precise={false} />}
        sub={t("{0, plural, one {# stop} other {# stops}} past the notice deadline", summary.lost)}
      />
      <KpiStripItem
        label={t("Longest wait")}
        value={formatDetentionMinutes(floor.longestOnSiteMinutes)}
        sub={floor.longestLocationName || t("Nothing on a dock")}
      />
    </KpiStrip>
  );
}
