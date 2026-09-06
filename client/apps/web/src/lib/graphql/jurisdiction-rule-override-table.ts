import { JurisdictionRuleOverrideTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const jurisdictionRuleOverrideTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: JurisdictionRuleOverrideTableDocument,
  operationName: "JurisdictionRuleOverrideTable",
  connectionKey: "jurisdictionRuleOverrides",
});
