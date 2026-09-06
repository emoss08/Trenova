import { HoldReasonTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const holdReasonTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: HoldReasonTableDocument,
  operationName: "HoldReasonTable",
  connectionKey: "holdReasons",
});
