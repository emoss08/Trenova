import { DataTable } from "@/components/data-table/data-table";
import { VariableFormatSchema } from "@/lib/schemas/variable-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./format-columns";
import { CreateFormatModal } from "./format-create-modal";
import { EditFormatModal } from "./format-edit-modal";

export function FormatDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<VariableFormatSchema>
      queryKey="format-list"
      name="Format"
      link="/variable-formats/"
      exportModelName="Format"
      columns={columns}
      resource={Resource.VariableFormat}
      TableModal={CreateFormatModal}
      TableEditModal={EditFormatModal}
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
