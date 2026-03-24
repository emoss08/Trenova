import { Package } from "lucide-react";
import { KPICard } from "../kpi-card";
import type { ShipmentAnalyticsData } from "../mock-data";

type Props = {
  data: ShipmentAnalyticsData["readyToDispatch"];
};

export function ReadyToDispatchCard({ data }: Props) {
  return (
    <KPICard
      label="Ready to Dispatch"
      value={data.count.toLocaleString()}
      icon={Package}
      detail="Awaiting dispatch"
      onClick={() => {}}
    />
  );
}
