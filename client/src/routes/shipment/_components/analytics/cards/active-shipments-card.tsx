import { TrendingDown, TrendingUp } from "lucide-react";
import { KPICard } from "../kpi-card";
import type { ShipmentAnalyticsData } from "../mock-data";
import { Truck } from "lucide-react";

type Props = {
  data: ShipmentAnalyticsData["activeShipments"];
};

export function ActiveShipmentsCard({ data }: Props) {
  const { count, changeFromYesterday } = data;
  const isPositive = changeFromYesterday >= 0;
  const TrendIcon = isPositive ? TrendingUp : TrendingDown;

  return (
    <KPICard
      label="Active Shipments"
      value={count.toLocaleString()}
      icon={Truck}
    >
      <div
        className={`flex items-center gap-1 text-[11px] ${isPositive ? "text-emerald-600 dark:text-emerald-400" : "text-red-600 dark:text-red-400"}`}
      >
        <TrendIcon className="size-3" />
        <span>
          {isPositive ? "+" : ""}
          {changeFromYesterday} vs yesterday
        </span>
      </div>
    </KPICard>
  );
}
