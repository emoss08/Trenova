import { DataTable } from "@/components/data-table/data-table";
import { orderTableGraphQLConfig } from "@/lib/graphql/order-table";
import type { Order } from "@/types/order";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./order-columns";
import { OrderPanel } from "./order-panel";

export default function OrderTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<Order>
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
