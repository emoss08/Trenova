import { DataTable } from "@/components/data-table/data-table";
import { type HazmatSegregationRuleSchema } from "@/lib/schemas/hazmat-segregation-rule-schema";
import { useMemo } from "react";
import { getColumns } from "./hazmat-segregation-rule-columns";
import { CreateHazmatSegregationRuleModal } from "./hazmat-segregation-rule-create-modal";
import { EditHazmatSegregationRuleModal } from "./hazmat-segregation-rule-edit-modal";

export default function HazmatSegregationRuleTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HazmatSegregationRuleSchema>
      name="Hazmat Segregation Rule"
      link="/hazmat-segregation-rules/"
      exportModelName="hazmat-segregation-rule"
      queryKey="hazmat-segregation-rule-list"
      TableModal={CreateHazmatSegregationRuleModal}
      TableEditModal={EditHazmatSegregationRuleModal}
      columns={columns}
    />
  );
}
