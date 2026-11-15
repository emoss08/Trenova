import {
  EmailProfileTableDocument,
} from "@trenova/graphql/generated/documents/email-profile/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const emailProfileTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: EmailProfileTableDocument,
  operationName: "EmailProfileTable",
  connectionKey: "emailProfiles",
});
