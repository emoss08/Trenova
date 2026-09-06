import { JurisdictionRuleTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const jurisdictionRuleTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: JurisdictionRuleTableDocument,
  operationName: "JurisdictionRuleTable",
  connectionKey: "jurisdictionRules",
});

export type JurisdictionRuleRow = DataTableConfigRow<typeof jurisdictionRuleTableGraphQLConfig>;
