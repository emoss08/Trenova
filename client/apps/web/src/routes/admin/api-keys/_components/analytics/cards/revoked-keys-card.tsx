import { KPICard } from "@/routes/shipment/_components/analytics/kpi-card";
import { ShieldOff } from "lucide-react";
import type { ApiKeyAnalyticsData } from "../mock-data";

type Props = {
  data: ApiKeyAnalyticsData["revokedKeys"];
};

export function RevokedKeysCard({ data }: Props) {
  const { count, percentOfTotal } = data;

  return (
    <KPICard
      label="Revoked Keys"
      value={count.toLocaleString()}
      icon={ShieldOff}
    >
      <div className="mt-1.5 space-y-1">
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full rounded-full bg-red-500 transition-all"
            style={{ width: `${Math.min(percentOfTotal, 100)}%` }}
          />
        </div>
        <div className="flex justify-between text-[10px] text-muted-foreground">
          <span>{percentOfTotal}% of total</span>
        </div>
      </div>
    </KPICard>
  );
}
