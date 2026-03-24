import { DataTable } from "@/components/data-table/data-table";
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
      link="/audit-entries/"
      queryKey="audit-entry-list"
      exportModelName="audit-entry"
      resource={Resource.AuditLog}
      columns={columns}
      TablePanel={AuditLogPanel}
    />
  );
}
