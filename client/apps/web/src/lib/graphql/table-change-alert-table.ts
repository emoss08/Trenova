import { TcaSubscriptionTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const tcaSubscriptionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: TcaSubscriptionTableDocument,
  operationName: "TCASubscriptionTable",
  connectionKey: "tcaSubscriptions",
});

export type TCASubscriptionRow = DataTableConfigRow<typeof tcaSubscriptionTableGraphQLConfig>;
