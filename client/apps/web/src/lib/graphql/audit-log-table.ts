import { AuditLogTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const auditLogTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AuditLogTableDocument,
  operationName: "AuditLogTable",
  connectionKey: "auditEntries",
});

export type AuditEntryRow = DataTableConfigRow<typeof auditLogTableGraphQLConfig>;
