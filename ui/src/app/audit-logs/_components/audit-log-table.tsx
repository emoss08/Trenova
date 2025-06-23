import { DataTable } from "@/components/data-table/data-table";
import { LiveModePresets } from "@/lib/live-mode-utils";
import { AuditEntry, Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./audit-log-columns";
import { AuditLogDetailsSheet } from "./audit-log-sheet";

export default function AuditLogTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AuditEntry>
      resource={Resource.AuditEntry}
      name="Audit Entry"
      link="/audit-logs/"
      includeOptions={false}
      queryKey="audit-log-list"
      exportModelName="audit-log"
      TableEditModal={AuditLogDetailsSheet}
      columns={columns}
      liveMode={LiveModePresets.auditLogs()}
    />
  );
}
