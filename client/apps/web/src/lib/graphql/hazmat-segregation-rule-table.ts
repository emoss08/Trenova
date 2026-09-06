import { HazmatSegregationRuleTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const hazmatSegregationRuleTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: HazmatSegregationRuleTableDocument,
  operationName: "HazmatSegregationRuleTable",
  connectionKey: "hazmatSegregationRules",
});
