import { KPICard } from "@/routes/shipment/_components/analytics/kpi-card";
import { KeyRound } from "lucide-react";
import type { ApiKeyAnalyticsData } from "../mock-data";

type Props = {
  data: ApiKeyAnalyticsData["totalKeys"];
};

export function TotalKeysCard({ data }: Props) {
  const { count, newThisMonth } = data;

  return (
    <KPICard
      label="Total Keys"
      value={count.toLocaleString()}
      icon={KeyRound}
      detail={`+${newThisMonth} this month`}
    />
  );
}
