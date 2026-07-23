import {
  TcaSubscriptionTableDocument,
  type TcaSubscriptionTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { TCASubscription } from "@/types/table-change-alert";

export const tcaSubscriptionTableGraphQLConfig = defineDataTableGraphQLConfig<
  TCASubscription,
  TcaSubscriptionTableQueryVariables
>({
  document: TcaSubscriptionTableDocument,
  operationName: "TCASubscriptionTable",
  connectionKey: "tcaSubscriptions",
});
