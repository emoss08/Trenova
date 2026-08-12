import { DataTable } from "@/components/data-table/data-table";
import {
  carrierCostEventTableGraphQLConfig,
  type CarrierCostEventRow,
} from "@/lib/graphql/carrier-settlement";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./cost-event-columns";
import { CostEventPanel } from "./cost-event-panel";

export default function CostEventsTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CarrierCostEventRow>
      name="Carrier Cost Event"
      queryKey="carrier-cost-event-list"
      graphql={carrierCostEventTableGraphQLConfig}
      resource={Resource.CarrierSettlement}
      columns={columns}
      TablePanel={CostEventPanel}
      enableCreateAction={false}
    />
  );
}
