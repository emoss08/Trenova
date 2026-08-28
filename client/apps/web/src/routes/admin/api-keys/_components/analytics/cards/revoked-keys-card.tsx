import { KPICard } from "@/components/kpi/kpi-simple-card";
import { ShieldOff } from "lucide-react";
import type { ApiKeyAnalyticsData } from "../mock-data";

type Props = {
  data: ApiKeyAnalyticsData["revokedKeys"];
};

export function RevokedKeysCard({ data }: Props) {
  const { count, percentOfTotal } = data;

  return (
    <KPICard label="Revoked Keys" value={count.toLocaleString()} icon={ShieldOff}>
      <div className="mt-1.5 space-y-1">
        <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
          <div
            className="h-full rounded-full bg-red-500 transition-all"
            style={{ width: `${Math.min(percentOfTotal, 100)}%` }}
          />
        </div>
        <div className="text-muted-foreground flex justify-between text-[10px]">
          <span>{percentOfTotal}% of total</span>
        </div>
      </div>
    </KPICard>
  );
}
