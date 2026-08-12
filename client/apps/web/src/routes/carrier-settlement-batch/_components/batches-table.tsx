import { DataTable } from "@/components/data-table/data-table";
import {
  carrierSettlementBatchTableGraphQLConfig,
  type CarrierSettlementBatchRow,
} from "@/lib/graphql/carrier-settlement";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./batch-columns";
import { CarrierBatchPanel } from "./batch-panel";

export default function CarrierBatchesTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CarrierSettlementBatchRow>
      name="Carrier Settlement Batch"
      queryKey="carrier-settlement-batch-list"
      graphql={carrierSettlementBatchTableGraphQLConfig}
      resource={Resource.CarrierSettlement}
      columns={columns}
      TablePanel={CarrierBatchPanel}
    />
  );
}
