import {
  DetentionPolicyTableDocument,
  type DetentionPolicyTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DetentionPolicy } from "@trenova/shared/types/detention";

export const detentionPolicyTableGraphQLConfig = defineDataTableGraphQLConfig<
  DetentionPolicy,
  DetentionPolicyTableQueryVariables
>({
  document: DetentionPolicyTableDocument,
  operationName: "DetentionPolicyTable",
  connectionKey: "detentionPolicies",
});
