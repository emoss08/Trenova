import { OrderTableDocument } from "@trenova/graphql/generated/documents/order/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const orderTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: OrderTableDocument,
  operationName: "OrderTable",
  connectionKey: "orders",
});

export type OrderRow = DataTableConfigRow<typeof orderTableGraphQLConfig>;
