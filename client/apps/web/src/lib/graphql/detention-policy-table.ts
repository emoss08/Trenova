import { DetentionPolicyTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const detentionPolicyTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DetentionPolicyTableDocument,
  operationName: "DetentionPolicyTable",
  connectionKey: "detentionPolicies",
});

export type DetentionPolicyRow = DataTableConfigRow<typeof detentionPolicyTableGraphQLConfig>;
