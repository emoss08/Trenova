import { Clock } from "lucide-react";
import { KPICard } from "../kpi-card";
import type { ShipmentAnalyticsData } from "../mock-data";

type Props = {
  data: ShipmentAnalyticsData["onTimePercent"];
};

export function OnTimeCard({ data }: Props) {
  const { percent, totalCount } = data;
  const target = 90;
  const hasData = totalCount > 0;
  const diff = Math.round((percent - target) * 10) / 10;
  const aboveTarget = diff >= 0;

  return (
    <KPICard
      label="On-Time %"
      value={hasData ? `${percent}%` : "—"}
      icon={Clock}
    >
      {hasData ? (
        <div className="mt-1.5 space-y-1">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-emerald-500 transition-all"
              style={{ width: `${Math.min(percent, 100)}%` }}
            />
          </div>
          <div className="flex justify-between text-[10px] text-muted-foreground">
            <span>Target: {target}%</span>
            <span
              className={
                aboveTarget
                  ? "text-emerald-600 dark:text-emerald-400"
                  : "text-red-600 dark:text-red-400"
              }
            >
              {aboveTarget ? "+" : ""}
              {diff}% {aboveTarget ? "above" : "below"}
            </span>
          </div>
        </div>
      ) : null}
    </KPICard>
  );
}
