import {
  JurisdictionRuleTableDocument,
  type JurisdictionRuleTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { JurisdictionRule } from "@/types/jurisdiction-rule";

export const jurisdictionRuleTableGraphQLConfig = defineDataTableGraphQLConfig<
  JurisdictionRule,
  JurisdictionRuleTableQueryVariables
>({
  document: JurisdictionRuleTableDocument,
  operationName: "JurisdictionRuleTable",
  connectionKey: "jurisdictionRules",
});
