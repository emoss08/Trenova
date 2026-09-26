import {
  AccountingDriftFindingTableDocument,
  type AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const ACCOUNTING_DRIFT_TABLE_KEY = "accounting-drift-findings";

export function createAccountingDriftTableGraphQLConfig(integrationType: AccountingSystem) {
  return defineDataTableGraphQLConfig({
    document: AccountingDriftFindingTableDocument,
    operationName: "AccountingDriftFindingTable",
    connectionKey: "accountingDriftFindingTable",
    extraVariables: { integrationType },
  });
}

export type AccountingDriftRow = DataTableConfigRow<
  ReturnType<typeof createAccountingDriftTableGraphQLConfig>
>;
