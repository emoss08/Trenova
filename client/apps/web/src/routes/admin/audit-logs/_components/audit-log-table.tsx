import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { auditLogTableGraphQLConfig, type AuditEntryRow } from "@/lib/graphql/audit-log-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./audit-log-columns";
import { AuditLogPanel } from "./audit-log-panel";

export default function AuditLogTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<AuditEntryRow>
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
