import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  jurisdictionRuleTableGraphQLConfig,
  type JurisdictionRuleRow,
} from "@/lib/graphql/jurisdiction-rule-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./jurisdiction-rule-columns";
import { JurisdictionRulePanel } from "./jurisdiction-rule-panel";

export default function JurisdictionRuleTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<JurisdictionRuleRow>
      name="Jurisdiction Rule"
      queryKey="jurisdiction-rule-list"
      graphql={jurisdictionRuleTableGraphQLConfig}
      resource={Resource.JurisdictionRule}
      columns={columns}
      TablePanel={JurisdictionRulePanel}
    />
  );
}
