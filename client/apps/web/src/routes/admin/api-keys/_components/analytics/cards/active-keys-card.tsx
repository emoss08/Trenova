import { useT } from "@trenova/shared/i18n/use-t";
import { KPICard } from "@/components/kpi/kpi-simple-card";
import { ShieldTickIcon } from "@trenova/shared/components/icons";
import type { ApiKeyAnalyticsData } from "../analytics-data";

type Props = {
  data: ApiKeyAnalyticsData["activeKeys"];
};

export function ActiveKeysCard({ data }: Props) {
  const t = useT();

  const { count, percentOfTotal } = data;

  return (
    <KPICard
      label={t("Active keys")}
      value={count.toLocaleString()}
      icon={ShieldTickIcon}
      detail={`${percentOfTotal}% of total`}
    />
  );
}
