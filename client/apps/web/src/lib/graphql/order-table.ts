import {
  OrderTableDocument,
  type OrderTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { Order } from "@trenova/shared/types/order";

export const orderTableGraphQLConfig = defineDataTableGraphQLConfig<
  Order,
  OrderTableQueryVariables
>({
  document: OrderTableDocument,
  operationName: "OrderTable",
  connectionKey: "orders",
});
