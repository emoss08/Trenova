import {
  HazmatSegregationRuleTableDocument,
} from "@trenova/graphql/generated/documents/hazmat-segregation-rule/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const hazmatSegregationRuleTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: HazmatSegregationRuleTableDocument,
  operationName: "HazmatSegregationRuleTable",
  connectionKey: "hazmatSegregationRules",
});
