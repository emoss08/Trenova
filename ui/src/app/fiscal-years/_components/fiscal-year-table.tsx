import { DataTable } from "@/components/data-table/data-table";
import { FiscalYearSchema } from "@/lib/schemas/fiscal-year-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./fiscal-year-columns";
import { CreateFiscalYearModal } from "./fiscal-year-create-modal";
import { EditFiscalYearModal } from "./fiscal-year-edit-modal";

export default function FiscalYearsDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<FiscalYearSchema>
      resource={Resource.FiscalYear}
      name="Fiscal Year"
      link="/fiscal-years/"
      queryKey="fiscal-year-list"
      exportModelName="fiscal-year"
      TableModal={CreateFiscalYearModal}
      TableEditModal={EditFiscalYearModal}
      columns={columns}
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
    />
  );
}
