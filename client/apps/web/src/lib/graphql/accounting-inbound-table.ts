import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import {
  AccountingInboundChangeTableDocument,
} from "@trenova/graphql/generated/documents/accounting-sync/inbound";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const ACCOUNTING_INBOUND_TABLE_KEY = "accounting-inbound-changes";

export function createAccountingInboundTableGraphQLConfig(integrationType: AccountingSystem) {
  return defineDataTableGraphQLConfig({
    document: AccountingInboundChangeTableDocument,
    operationName: "AccountingInboundChangeTable",
    connectionKey: "accountingInboundChangeTable",
    extraVariables: { integrationType },
  });
}

export type AccountingInboundRow = DataTableConfigRow<
  ReturnType<typeof createAccountingInboundTableGraphQLConfig>
>;
