import { DataTable } from "@/components/data-table/data-table";
import {
  settlementBatchTableGraphQLConfig,
  type SettlementBatchRow,
} from "@/lib/graphql/driver-settlement";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./batch-columns";
import { BatchPanel } from "./batch-panel";

export default function BatchesTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<SettlementBatchRow>
      name="Settlement Batch"
      queryKey="settlement-batch-list"
      graphql={settlementBatchTableGraphQLConfig}
      resource={Resource.DriverSettlement}
      columns={columns}
      TablePanel={BatchPanel}
    />
  );
}
