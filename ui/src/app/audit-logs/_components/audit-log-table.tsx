import { DataTable } from "@/components/data-table/data-table";
import { AuditEntry } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./audit-log-columns";
import { AuditLogDetailsSheet } from "./audit-log-sheet";

export default function AuditLogTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AuditEntry>
      name="Audit Log"
      link="/audit-logs/"
      includeOptions={false}
      queryKey="audit-log-list"
      exportModelName="audit-log"
      TableEditModal={AuditLogDetailsSheet}
      defaultSort={[{ id: "timestamp", desc: true }]}
      columns={columns}
    />
  );
}
