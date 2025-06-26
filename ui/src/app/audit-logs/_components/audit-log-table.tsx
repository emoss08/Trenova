import { DataTableV2 } from "@/components/data-table/data-table-v2";
import { LiveModePresets } from "@/lib/live-mode-utils";
import { AuditEntry, Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./audit-log-columns";
import { AuditLogDetailsSheet } from "./audit-log-sheet";

export default function AuditLogTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTableV2<AuditEntry>
      resource={Resource.AuditEntry}
      name="Audit Entry"
      link="/audit-logs/"
      includeOptions={false}
      queryKey="audit-log-list"
      exportModelName="audit-log"
      TableEditModal={AuditLogDetailsSheet}
      columns={columns}
      useEnhancedBackend={true}
      liveMode={LiveModePresets.auditLogs()}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
    />
  );
}
