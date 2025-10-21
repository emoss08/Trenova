import { DataTable } from "@/components/data-table/data-table";
import type { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./service-type-columns";
import { CreateServiceTypeModal } from "./service-type-create-modal";
import { EditServiceTypeModal } from "./service-type-edit-modal";

export default function ServiceTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<ServiceTypeSchema>
      resource={Resource.ServiceType}
      name="Service Type"
      link="/service-types/"
      queryKey="service-type-list"
      exportModelName="service-type"
      TableModal={CreateServiceTypeModal}
      TableEditModal={EditServiceTypeModal}
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
