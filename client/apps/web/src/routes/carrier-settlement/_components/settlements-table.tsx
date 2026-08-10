import { DataTable } from "@/components/data-table/data-table";
import {
  carrierSettlementTableGraphQLConfig,
  type CarrierSettlementRow,
} from "@/lib/graphql/carrier-settlement";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./settlement-columns";
import { CarrierSettlementPanel } from "./settlement-panel";

/**
 * Read-only settlement history. Lifecycle mutations (submit, approve, post,
 * mark paid) live in the carrier settlement workspace — this table only reads,
 * filters, and exports, matching the page's own description.
 */
export default function CarrierSettlementsTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CarrierSettlementRow>
      name="Carrier Settlement"
      queryKey="carrier-settlement-list"
      graphql={carrierSettlementTableGraphQLConfig}
      resource={Resource.CarrierSettlement}
      columns={columns}
      TablePanel={CarrierSettlementPanel}
      enableCreateAction={false}
    />
  );
}
