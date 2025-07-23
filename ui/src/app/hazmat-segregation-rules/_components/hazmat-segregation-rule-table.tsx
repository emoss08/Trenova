/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { type HazmatSegregationRuleSchema } from "@/lib/schemas/hazmat-segregation-rule-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./hazmat-segregation-rule-columns";
import { CreateHazmatSegregationRuleModal } from "./hazmat-segregation-rule-create-modal";
import { EditHazmatSegregationRuleModal } from "./hazmat-segregation-rule-edit-modal";

export default function HazmatSegregationRuleTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HazmatSegregationRuleSchema>
      resource={Resource.HazmatSegregationRule}
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
