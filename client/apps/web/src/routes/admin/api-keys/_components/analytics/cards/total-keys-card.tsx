import { useT } from "@trenova/shared/i18n/use-t";
import { KPICard } from "@/components/kpi/kpi-simple-card";
import { KeyRound } from "lucide-react";
import type { ApiKeyAnalyticsData } from "../analytics-data";

type Props = {
  data: ApiKeyAnalyticsData["totalKeys"];
};

export function TotalKeysCard({ data }: Props) {
  const t = useT();

  const { count, newThisMonth } = data;

  return (
    <KPICard
      label={t("Total Keys")}
      value={count.toLocaleString()}
      icon={KeyRound}
      detail={`+${newThisMonth} this month`}
    />
  );
}
