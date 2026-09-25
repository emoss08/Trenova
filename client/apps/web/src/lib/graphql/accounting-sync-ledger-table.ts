import {
  AccountingSyncRecordTableDocument,
  type AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const ACCOUNTING_SYNC_LEDGER_KEY = "accounting-sync-ledger";

export function createAccountingSyncLedgerTableGraphQLConfig(integrationType: AccountingSystem) {
  return defineDataTableGraphQLConfig({
    document: AccountingSyncRecordTableDocument,
    operationName: "AccountingSyncRecordTable",
    connectionKey: "accountingSyncRecordTable",
    extraVariables: { integrationType },
  });
}

export type AccountingSyncLedgerRow = DataTableConfigRow<
  ReturnType<typeof createAccountingSyncLedgerTableGraphQLConfig>
>;
