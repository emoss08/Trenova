import { DataTable } from "@/components/data-table/data-table";
import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./fleet-code-columns";
import { CreateFleetCodeModal } from "./fleet-code-create-modal";
import { EditFleetCodeModal } from "./fleet-code-edit-modal";

export default function FleetCodesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<FleetCodeSchema>
      name="Fleet Code"
      link="/fleet-codes/"
      queryKey="fleet-code-list"
      extraSearchParams={{
        includeManagerDetails: true,
      }}
      exportModelName="fleet-code"
      TableModal={CreateFleetCodeModal}
      TableEditModal={EditFleetCodeModal}
      columns={columns}
      resource={Resource.FleetCode}
      defaultSort={[{ field: "createdAt", direction: "desc" }]}
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
