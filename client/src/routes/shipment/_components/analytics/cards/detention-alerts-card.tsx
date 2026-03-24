import { AlertTriangle } from "lucide-react";
import { KPICard } from "../kpi-card";
import type { ShipmentAnalyticsData } from "../mock-data";

type Props = {
  data: ShipmentAnalyticsData["detentionAlerts"];
};

export function DetentionAlertsCard({ data }: Props) {
  const { count } = data;

  return (
    <KPICard
      label="Detention Alerts"
      value={count.toLocaleString()}
      icon={AlertTriangle}
      detail={`${count} active ${count === 1 ? "alert" : "alerts"}`}
    />
  );
}
