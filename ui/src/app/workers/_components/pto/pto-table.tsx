import { DataTable } from "@/components/data-table/data-table";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./pto-columns";
import { PTOCreateModal } from "./pto-create-modal";
import { EditPTOModal } from "./pto-edit-modal";

export function PtoDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<WorkerPTOSchema>
      queryKey="worker-pto-list"
      name="Worker PTO"
      link="/workers/pto/"
      exportModelName="Worker PTO"
      columns={columns}
      resource={Resource.WorkerPTO}
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
      TableModal={PTOCreateModal}
      TableEditModal={EditPTOModal}
      defaultSort={[{ field: "createdAt", direction: "desc" }]}
    />
  );
}
