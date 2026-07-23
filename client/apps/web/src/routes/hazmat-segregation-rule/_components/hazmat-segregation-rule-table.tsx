import { DataTable } from "@/components/data-table/data-table";
import { hazmatSegregationRuleTableGraphQLConfig } from "@/lib/graphql/hazmat-segregation-rule-table";
import { Resource } from "@trenova/shared/types/permission";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";
import { useMemo } from "react";
import { getColumns } from "./hazmat-segregation-rule-columns";
import { HazmatSegregationRulePanel } from "./hazmat-segregation-rule-panel";

export default function HazmatSegregationRuleTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HazmatSegregationRule>
      name="Hazmat Segregation Rule"
      queryKey="hazmat-segregation-rule-list"
      graphql={hazmatSegregationRuleTableGraphQLConfig}
      resource={Resource.HazmatSegregationRule}
      columns={columns}
      TablePanel={HazmatSegregationRulePanel}
    />
  );
}
