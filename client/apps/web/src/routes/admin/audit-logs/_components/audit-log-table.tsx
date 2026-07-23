import { DataTable } from "@/components/data-table/data-table";
import { auditLogTableGraphQLConfig } from "@/lib/graphql/audit-log-table";
import type { AuditEntry } from "@/types/audit-entry";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./audit-log-columns";
import { AuditLogPanel } from "./audit-log-panel";

export default function AuditLogTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AuditEntry>
      name="Audit Entry"
      queryKey="audit-entry-list"
      graphql={auditLogTableGraphQLConfig}
      resource={Resource.AuditLog}
      columns={columns}
      TablePanel={AuditLogPanel}
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}
