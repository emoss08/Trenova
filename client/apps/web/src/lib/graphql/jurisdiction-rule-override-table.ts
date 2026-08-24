import {
  JurisdictionRuleOverrideTableDocument,
  type JurisdictionRuleOverrideTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";

export const jurisdictionRuleOverrideTableGraphQLConfig = defineDataTableGraphQLConfig<
  JurisdictionRuleOverride,
  JurisdictionRuleOverrideTableQueryVariables
>({
  document: JurisdictionRuleOverrideTableDocument,
  operationName: "JurisdictionRuleOverrideTable",
  connectionKey: "jurisdictionRuleOverrides",
});
