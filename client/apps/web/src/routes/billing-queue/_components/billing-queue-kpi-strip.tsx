import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";

type StatMetric = {
  key: string;
  label: string;
  value: number;
};

export function BillingQueueKPIStrip({
  statusFilter,
  includePosted,
  onFilterChange,
}: {
  statusFilter: string | null;
  includePosted: boolean;
  onFilterChange: (status: string | null) => void;
}) {
  const t = useT();

  const { data: stats } = useQuery(queries.billingQueue.stats());

  const toggle = (status: string) => {
    onFilterChange(statusFilter === status ? null : status);
  };

  const metrics: StatMetric[] = [
    {
      key: "ReadyForReview",
      label: t("Pending"),
      value: stats?.readyForReview ?? 0,
    },
    {
      key: "InReview",
      label: t("In review"),
      value: stats?.inReview ?? 0,
    },
    {
      key: "Exception",
      label: t("Exceptions"),
      value: (stats?.onHold ?? 0) + (stats?.exception ?? 0) + (stats?.sentBackToOps ?? 0),
    },
    {
      key: "Approved",
      label: includePosted ? "Approved drafts" : "Approved",
      value: stats?.approved ?? 0,
    },
  ];

  return (
    <KpiStrip>
      {metrics.map((metric) => (
        <KpiStripItem
          key={metric.key}
          label={t(metric.label)}
          value={metric.value}
          active={statusFilter === metric.key}
          onClick={() => toggle(metric.key)}
        />
      ))}
    </KpiStrip>
  );
}
