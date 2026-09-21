import { useT } from "@trenova/shared/i18n/use-t";
import {
  AgingDistributionBar,
  type AgingBucketTotals,
} from "@/components/accounting/aging-buckets";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { SectionPanel } from "@/components/section-panel";
import { AR_DSO_TARGET_DAYS } from "@/lib/accounting-constants";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";

export function AgingSummaryHeader({
  totals,
  isLoading,
}: {
  totals: AgingBucketTotals | undefined;
  isLoading: boolean;
}) {
  const t = useT();

  const { data: kpis } = useQuery(queries.ar.dashboardKpis());

  if (isLoading || !totals) {
    return <KpiStripSkeleton count={5} />;
  }

  const totalOpen = totals.totalOpenMinor;
  const currentShare = totalOpen > 0 ? (totals.currentMinor / totalOpen) * 100 : 0;
  const overdueShare = totalOpen > 0 ? 100 - currentShare : 0;

  return (
    <div className="space-y-3">
      <KpiStrip>
        <KpiStripItem label={t("Total open")} value={formatCurrency(totalOpen / 100)} />
        <KpiStripItem
          label={t("Current")}
          value={`${currentShare.toFixed(1)}%`}
          sub={formatCurrency(totals.currentMinor / 100)}
          tone="success"
        />
        <KpiStripItem
          label={t("Overdue")}
          value={`${overdueShare.toFixed(1)}%`}
          sub={formatCurrency((totalOpen - totals.currentMinor) / 100)}
          tone={overdueShare > 0 ? "danger" : undefined}
        />
        <KpiStripItem
          label={t("Current DSO")}
          value={kpis ? `${kpis.currentDsoDays.toFixed(1)}d` : "—"}
          sub={`target < ${AR_DSO_TARGET_DAYS}d`}
        />
        <KpiStripItem
          label={t("CEI")}
          value={kpis ? `${kpis.cei.toFixed(0)}%` : "—"}
          sub={t("trailing 90 days")}
        />
      </KpiStrip>

      {totalOpen > 0 ? (
        <SectionPanel title={t("Distribution")}>
          <div className="px-3 py-3">
            <AgingDistributionBar totals={totals} />
          </div>
        </SectionPanel>
      ) : null}
    </div>
  );
}
