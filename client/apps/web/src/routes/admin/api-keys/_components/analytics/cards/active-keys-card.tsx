import { KPICard } from "@/components/kpi/kpi-simple-card";
import { ShieldCheck } from "lucide-react";
import type { ApiKeyAnalyticsData } from "../analytics-data";

type Props = {
  data: ApiKeyAnalyticsData["activeKeys"];
};

export function ActiveKeysCard({ data }: Props) {
  const { count, percentOfTotal } = data;

  return (
    <KPICard
      label="Active Keys"
      value={count.toLocaleString()}
      icon={ShieldCheck}
      detail={`${percentOfTotal}% of total`}
    />
  );
}
