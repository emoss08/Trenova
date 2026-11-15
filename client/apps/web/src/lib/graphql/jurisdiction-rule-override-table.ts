import {
  JurisdictionRuleOverrideTableDocument,
} from "@trenova/graphql/generated/documents/jurisdiction-rule/overrides";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const jurisdictionRuleOverrideTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: JurisdictionRuleOverrideTableDocument,
  operationName: "JurisdictionRuleOverrideTable",
  connectionKey: "jurisdictionRuleOverrides",
});
