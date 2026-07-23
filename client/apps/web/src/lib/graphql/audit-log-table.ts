import {
  AuditLogTableDocument,
  type AuditLogTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { AuditEntry } from "@/types/audit-entry";

export const auditLogTableGraphQLConfig = defineDataTableGraphQLConfig<
  AuditEntry,
  AuditLogTableQueryVariables
>({
  document: AuditLogTableDocument,
  operationName: "AuditLogTable",
  connectionKey: "auditEntries",
});
