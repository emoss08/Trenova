import {
  EmailProfileTableDocument,
  type EmailProfileTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { EmailProfile } from "@/types/email";

export const emailProfileTableGraphQLConfig = defineDataTableGraphQLConfig<
  EmailProfile,
  EmailProfileTableQueryVariables
>({
  document: EmailProfileTableDocument,
  operationName: "EmailProfileTable",
  connectionKey: "emailProfiles",
});
