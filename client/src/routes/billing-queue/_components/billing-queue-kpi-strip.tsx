import { KPICard } from "@/routes/shipment/_components/analytics/kpi-card";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  EyeIcon,
  InboxIcon,
} from "lucide-react";

export function BillingQueueKPIStrip({
  statusFilter,
  onFilterChange,
}: {
  statusFilter: string | null;
  onFilterChange: (status: string | null) => void;
}) {
  const { data: stats } = useQuery(queries.billingQueue.stats());

  const toggle = (status: string) => {
    onFilterChange(statusFilter === status ? null : status);
  };

  return (
    <div className="grid grid-cols-4 gap-3 px-4 pt-3">
      <KPICard
        label="Pending Review"
        value={String(stats?.readyForReview ?? 0)}
        icon={InboxIcon}
        detail="Awaiting biller pickup"
        onClick={() => toggle("ReadyForReview")}
      />
      <KPICard
        label="In Review"
        value={String(stats?.inReview ?? 0)}
        icon={EyeIcon}
        detail="Currently being reviewed"
        onClick={() => toggle("InReview")}
      />
      <KPICard
        label="Exceptions"
        value={String(
          (stats?.onHold ?? 0) + (stats?.exception ?? 0) + (stats?.sentBackToOps ?? 0),
        )}
        icon={AlertTriangleIcon}
        detail="Needs attention"
        onClick={() => toggle("Exception")}
      />
      <KPICard
        label="Approved"
        value={String(stats?.approved ?? 0)}
        icon={CheckCircleIcon}
        detail="Ready for invoicing"
        onClick={() => toggle("Approved")}
      />
    </div>
  );
}
