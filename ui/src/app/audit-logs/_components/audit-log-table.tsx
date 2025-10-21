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
      link="/audit-entries/"
      includeOptions={false}
      queryKey="audit-entry-list"
      exportModelName="audit-entry"
      TableEditModal={AuditLogDetailsSheet}
      columns={columns}
      useEnhancedBackend={true}
      liveMode={LiveModePresets.auditEntries()}
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
