import { cn } from "@/lib/utils";
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
  const { data: stats } = useQuery(queries.billingQueue.stats());

  const toggle = (status: string) => {
    onFilterChange(statusFilter === status ? null : status);
  };

  const metrics: StatMetric[] = [
    {
      key: "ReadyForReview",
      label: "Pending",
      value: stats?.readyForReview ?? 0,
    },
    {
      key: "InReview",
      label: "In Review",
      value: stats?.inReview ?? 0,
    },
    {
      key: "Exception",
      label: "Exceptions",
      value:
        (stats?.onHold ?? 0) +
        (stats?.exception ?? 0) +
        (stats?.sentBackToOps ?? 0),
    },
    {
      key: "Approved",
      label: includePosted ? "Approved Drafts" : "Approved",
      value: stats?.approved ?? 0,
    },
  ];

  return (
    <div className="mx-4 mt-3">
      <div className="flex items-center rounded-lg border bg-card">
        {metrics.map((metric, index) => (
          <button
            key={metric.key}
            type="button"
            onClick={() => toggle(metric.key)}
            className={cn(
              "flex items-center gap-2 px-4 py-2.5 text-left transition-colors",
              "hover:bg-muted/50",
              index > 0 && "border-l",
              statusFilter === metric.key && "bg-muted",
            )}
          >
            <span className="text-xs text-muted-foreground">{metric.label}</span>
            <span className="text-sm font-semibold tabular-nums">{metric.value}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
