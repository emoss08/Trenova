import { DataTable } from "@/components/data-table/data-table";
import { Resource } from "@/types/permission";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";
import { useMemo } from "react";
import { getColumns } from "./hazmat-segregation-rule-columns";
import { HazmatSegregationRulePanel } from "./hazmat-segregation-rule-panel";

export default function HazmatSegregationRuleTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HazmatSegregationRule>
      name="Hazmat Segregation Rule"
      link="/hazmat-segregation-rules/"
      queryKey="hazmat-segregation-rule-list"
      exportModelName="hazmat-segregation-rule"
      resource={Resource.HazmatSegregationRule}
      columns={columns}
      TablePanel={HazmatSegregationRulePanel}
    />
  );
}
