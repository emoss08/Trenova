import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { MetricSkeleton } from "@/components/metric-skeleton";
import type { ApprovedPTOMetrics } from "./approved-pto-metrics";

export function ApprovedPTOKPICards({
  metrics,
  requestedCount,
  chartLoading,
  requestedLoading,
  requestedError,
}: {
  metrics: ApprovedPTOMetrics;
  requestedCount: number;
  chartLoading: boolean;
  requestedLoading: boolean;
  requestedError: boolean;
}) {
  const t = useT();

  if (chartLoading && requestedLoading) {
    return <MetricSkeleton />;
  }

  return (
    <div className="mb-3 flex shrink-0 flex-col gap-2">
      <KpiStrip>
        <KpiStripItem
          label={t("Approved PTO days")}
          value={metrics.approvedPtoDays.toLocaleString()}
          sub={t("Daily occupancy total")}
        />
        <KpiStripItem
          label={t("Requested PTO requests")}
          value={requestedError ? "--" : requestedCount.toLocaleString()}
          sub={t("Pending approvals in range")}
        />
        <KpiStripItem
          label={t("Workers with approved PTO")}
          value={metrics.workersWithApprovedPTO.toLocaleString()}
          sub={t("Unique workers in range")}
        />
        <KpiStripItem
          label={t("Peak day occupancy")}
          value={metrics.peakDay.occupancy.toLocaleString()}
          sub={metrics.peakDay.dateLabel ?? "No peak day"}
        />
      </KpiStrip>
      {requestedError && !requestedLoading && (
        <p className="border-border text-muted-foreground rounded-md border border-dashed px-2.5 py-2 text-xs">
          {t("Requested PTO metric is temporarily unavailable.")}
        </p>
      )}
    </div>
  );
}
