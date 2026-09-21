import { useT } from "@trenova/shared/i18n/use-t";
import { KPICard } from "@/components/kpi/kpi-simple-card";
import { ShieldOff } from "lucide-react";
import type { ApiKeyAnalyticsData } from "../analytics-data";

type Props = {
  data: ApiKeyAnalyticsData["revokedKeys"];
};

export function RevokedKeysCard({ data }: Props) {
  const t = useT();

  const { count, percentOfTotal } = data;

  return (
    <KPICard label={t("Revoked keys")} value={count.toLocaleString()} icon={ShieldOff}>
      <div className="mt-1.5 space-y-1">
        <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
          <div
            className="h-full rounded-full bg-danger transition-all"
            style={{ width: `${Math.min(percentOfTotal, 100)}%` }}
          />
        </div>
        <div className="text-muted-foreground flex justify-between text-2xs">
          <span>{t("{0}% of total", percentOfTotal)}</span>
        </div>
      </div>
    </KPICard>
  );
}
