import { DataTable } from "@/components/data-table/data-table";
import { riskDecisionTableGraphQLConfig, type RiskDecisionRow } from "@/lib/graphql/security-activity-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { riskDecisionColumns } from "./activity-columns";

export default function RiskDecisionsTable() {
  const t = useT();
  const columns = useMemo(() => riskDecisionColumns(t), [t]);

  return (
    <DataTable<RiskDecisionRow>
      name="Risk Decision"
      emptyTitle={t("No risk decisions recorded yet")}
      queryKey="security-risk-decisions"
      graphql={riskDecisionTableGraphQLConfig}
      resource={Resource.RiskDecision}
      columns={columns}
      enableCreateAction={false}
    />
  );
}
