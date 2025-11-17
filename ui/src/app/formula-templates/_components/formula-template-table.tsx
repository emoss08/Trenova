import { DataTable } from "@/components/data-table/data-table";
import { type FormulaTemplateSchema } from "@/lib/schemas/formula-template-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./formula-template-columns";
import { CreateFormulaTemplateModal } from "./formula-template-create-modal";
import { EditFormulaTemplateModal } from "./formula-template-edit-modal";

export default function FormulaTemplatesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<FormulaTemplateSchema>
      name="Formula Template"
      resource={Resource.FormulaTemplate}
      columns={columns}
      link="/formula-templates/"
      queryKey="formulaTemplates"
      exportModelName="FormulaTemplate"
      TableModal={CreateFormulaTemplateModal}
      TableEditModal={EditFormulaTemplateModal}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      useEnhancedBackend={true}
      defaultSort={[{ field: "createdAt", direction: "desc" }]}
    />
  );
}
