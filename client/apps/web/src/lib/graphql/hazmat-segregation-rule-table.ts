import {
  HazmatSegregationRuleTableDocument,
  type HazmatSegregationRuleTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";

export const hazmatSegregationRuleTableGraphQLConfig = defineDataTableGraphQLConfig<
  HazmatSegregationRule,
  HazmatSegregationRuleTableQueryVariables
>({
  document: HazmatSegregationRuleTableDocument,
  operationName: "HazmatSegregationRuleTable",
  connectionKey: "hazmatSegregationRules",
});
