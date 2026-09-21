import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import type { Tone } from "@/components/kpi/tone";
import {
  AR_CEI_HEALTHY_THRESHOLD,
  AR_CEI_WARNING_THRESHOLD,
  AR_DSO_TARGET_DAYS,
} from "@/lib/accounting-constants";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";

const AR_KPI_COUNT = 5;
const AR_KPI_MIN_WIDTH = "11rem";

function ceiTone(cei: number): Tone {
  if (cei >= AR_CEI_HEALTHY_THRESHOLD) return "success";
  if (cei >= AR_CEI_WARNING_THRESHOLD) return "warning";
  return "danger";
}

export function ARKpiRow() {
  const t = useT();

  const { data: kpis, isLoading } = useQuery(queries.ar.dashboardKpis());

  if (isLoading || !kpis) {
    return <KpiStripSkeleton count={AR_KPI_COUNT} size="lg" minItemWidth={AR_KPI_MIN_WIDTH} />;
  }

  const dsoDelta = kpis.dsoDeltaDays;
  const dsoMoved = Math.abs(dsoDelta) > 0.05;

  return (
    <KpiStrip minItemWidth={AR_KPI_MIN_WIDTH}>
      <KpiStripItem
        size="lg"
        label={t("AR outstanding")}
        value={formatCurrency(kpis.overview.totalOpenMinor / 100)}
        sub={`${kpis.overview.openInvoiceCount} open ${
          kpis.overview.openInvoiceCount === 1 ? "invoice" : "invoices"
        }`}
        to="/accounting/ar/aging"
      />
      <KpiStripItem
        size="lg"
        label={t("Days sales outstanding")}
        tone={kpis.currentDsoDays > AR_DSO_TARGET_DAYS ? "danger" : undefined}
        value={t("{0}d", kpis.currentDsoDays.toFixed(1))}
        delta={dsoMoved ? Number(dsoDelta.toFixed(1)) : null}
        deltaLabel="d"
        deltaTone={dsoDelta > 0 ? "danger" : "success"}
        sub={t("target < {0}d · vs 4 weeks ago", AR_DSO_TARGET_DAYS)}
      />
      <KpiStripItem
        size="lg"
        label={t("Collection effectiveness")}
        tone={ceiTone(kpis.cei)}
        value={`${kpis.cei.toFixed(0)}%`}
      />
      <KpiStripItem
        size="lg"
        label={t("Unapplied cash")}
        value={formatCurrency(kpis.overview.unappliedCashMinor / 100)}
        sub={t("awaiting application")}
        to="/accounting/ar/payments"
      />
      <KpiStripItem
        size="lg"
        label={t("Overdue")}
        tone={kpis.overduePercent >= 25 ? "danger" : undefined}
        value={`${kpis.overduePercent.toFixed(1)}%`}
        sub={`${formatCurrency(kpis.overview.overdueMinor / 100)} past due`}
        to="/accounting/ar/open-items"
      />
    </KpiStrip>
  );
}
