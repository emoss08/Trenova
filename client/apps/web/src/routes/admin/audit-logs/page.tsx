import { AuditAlert } from "@/components/audit-alert";
import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const AuditLogTable = lazy(() => import("./_components/audit-log-table"));

export function AuditLogsPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Audit Entries"
        description="Monitor and review system activity across your organization"
      />
      <div className="p-4">
        <AuditAlert />
        <DataTableLazyComponent>
          <AuditLogTable />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
