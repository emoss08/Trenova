import { OrderTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const orderTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: OrderTableDocument,
  operationName: "OrderTable",
  connectionKey: "orders",
});

export type OrderRow = DataTableConfigRow<typeof orderTableGraphQLConfig>;
