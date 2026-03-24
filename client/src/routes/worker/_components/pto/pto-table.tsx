import { DataTable } from "@/components/data-table/data-table";
import { Resource } from "@/types/permission";
import type { WorkerPTO } from "@/types/worker";
import { useMemo } from "react";
import { getColumns } from "./pto-columns";

export default function PTODataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<WorkerPTO>
      queryKey="worker-pto-list"
      name="Worker PTO"
      link="/worker-pto/"
      exportModelName="worker-pto"
      resource={Resource.WorkerPTO}
      columns={columns}
      enableRowSelection
      extraSearchParams={{
        includeWorker: true,
      }}
    />
  );
}
