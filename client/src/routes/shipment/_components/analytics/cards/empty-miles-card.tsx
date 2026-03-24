import { Gauge } from "lucide-react";
import { KPICard } from "../kpi-card";
import type { ShipmentAnalyticsData } from "../mock-data";

type Props = {
  data: ShipmentAnalyticsData["emptyMilePercent"];
};

export function EmptyMilesCard({ data }: Props) {
  const { percent, totalMiles } = data;
  const target = 15;
  const hasData = totalMiles > 0;
  const withinTarget = percent <= target;

  return (
    <KPICard
      label="Empty Mile %"
      value={hasData ? `${percent}%` : "—"}
      icon={Gauge}
    >
      {hasData ? (
        <div className="mt-1.5 space-y-1">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-emerald-500 transition-all"
              style={{ width: `${(Math.min(percent, 50) / 50) * 100}%` }}
            />
          </div>
          <div className="flex justify-between text-[10px] text-muted-foreground">
            <span>Target: &lt;{target}%</span>
            <span
              className={
                withinTarget
                  ? "text-emerald-600 dark:text-emerald-400"
                  : "text-red-600 dark:text-red-400"
              }
            >
              {withinTarget ? "Within target" : "Above target"}
            </span>
          </div>
        </div>
      ) : null}
    </KPICard>
  );
}
