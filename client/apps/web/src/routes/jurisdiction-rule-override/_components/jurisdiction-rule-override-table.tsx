import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { jurisdictionRuleOverrideTableGraphQLConfig } from "@/lib/graphql/jurisdiction-rule-override-table";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./jurisdiction-rule-override-columns";
import { JurisdictionRuleOverridePanel } from "./jurisdiction-rule-override-panel";

export default function JurisdictionRuleOverrideTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<JurisdictionRuleOverride>
      name="Carrier Override"
      queryKey="jurisdiction-rule-override-list"
      graphql={jurisdictionRuleOverrideTableGraphQLConfig}
      resource={Resource.JurisdictionRuleOverride}
      columns={columns}
      TablePanel={JurisdictionRuleOverridePanel}
    />
  );
}
