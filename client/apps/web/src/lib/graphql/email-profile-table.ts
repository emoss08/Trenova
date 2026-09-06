import { EmailProfileTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const emailProfileTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: EmailProfileTableDocument,
  operationName: "EmailProfileTable",
  connectionKey: "emailProfiles",
});
