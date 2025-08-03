/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
