import { DataTable } from "@/components/data-table/data-table";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { CreateWorkerModal } from "./workers-create-modal";
import { EditWorkerModal } from "./workers-edit-modal";
import { getColumns } from "./workers-table-columns";

export default function WorkersDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<WorkerSchema>
      extraSearchParams={{
        includeProfile: "true",
        includePTO: "true",
      }}
      TableModal={CreateWorkerModal}
      TableEditModal={EditWorkerModal}
      queryKey="worker-list"
      name="Worker"
      link="/workers/"
      exportModelName="Worker"
      columns={columns}
      resource={Resource.Worker}
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
