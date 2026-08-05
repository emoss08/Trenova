import { DataTable } from "@/components/data-table/data-table";
import { jurisdictionRuleTableGraphQLConfig } from "@/lib/graphql/jurisdiction-rule-table";
import type { JurisdictionRule } from "@/types/jurisdiction-rule";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./jurisdiction-rule-columns";
import { JurisdictionRulePanel } from "./jurisdiction-rule-panel";

export default function JurisdictionRuleTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<JurisdictionRule>
      name="Jurisdiction Rule"
      queryKey="jurisdiction-rule-list"
      graphql={jurisdictionRuleTableGraphQLConfig}
      resource={Resource.JurisdictionRule}
      columns={columns}
      TablePanel={JurisdictionRulePanel}
    />
  );
}
