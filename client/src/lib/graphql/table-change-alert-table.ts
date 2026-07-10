import {
  TcaSubscriptionTableDocument,
  type TcaSubscriptionTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { TCASubscription } from "@/types/table-change-alert";

export const tcaSubscriptionTableGraphQLConfig = defineDataTableGraphQLConfig<
  TCASubscription,
  TcaSubscriptionTableQueryVariables
>({
  document: TcaSubscriptionTableDocument,
  operationName: "TCASubscriptionTable",
  connectionKey: "tcaSubscriptions",
});
