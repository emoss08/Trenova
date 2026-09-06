import { DataTable } from "@/components/data-table/data-table";
import { orderTableGraphQLConfig, type OrderRow } from "@/lib/graphql/order-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./order-columns";
import { OrderPanel } from "./order-panel";

export default function OrderTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<OrderRow>
      name="Order"
      queryKey="order-list"
      graphql={orderTableGraphQLConfig}
      resource={Resource.Order}
      columns={columns}
      TablePanel={OrderPanel}
      enableRowSelection
    />
  );
}
