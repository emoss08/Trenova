import { useT } from "@trenova/shared/i18n/use-t";
import { KPICard } from "@/components/kpi/kpi-simple-card";
import { Key01Icon } from "@trenova/shared/components/icons";
import type { ApiKeyAnalyticsData } from "../analytics-data";

type Props = {
  data: ApiKeyAnalyticsData["totalKeys"];
};

export function TotalKeysCard({ data }: Props) {
  const t = useT();

  const { count, newThisMonth } = data;

  return (
    <KPICard
      label={t("Total keys")}
      value={count.toLocaleString()}
      icon={Key01Icon}
      detail={t("+{0} this month", newThisMonth)}
    />
  );
}
