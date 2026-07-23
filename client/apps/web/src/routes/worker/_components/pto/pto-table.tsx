import { DataTable } from "@/components/data-table/data-table";
import { workerTableGraphQLConfigs } from "@/lib/graphql/worker-table";
import { Resource } from "@trenova/shared/types/permission";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useMemo } from "react";
import { getColumns } from "./pto-columns";

export default function PTODataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<WorkerPTO>
      queryKey="worker-pto-list"
      name="Worker PTO"
      resource={Resource.WorkerPTO}
      columns={columns}
      graphql={workerTableGraphQLConfigs.pto}
      enableRowSelection
    />
  );
}
