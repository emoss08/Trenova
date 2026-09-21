import { useT } from "@trenova/shared/i18n/use-t";
import { AuditAlert } from "@/components/audit-alert";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const AuditLogTable = lazy(() => import("./_components/audit-log-table"));

export function AuditLogsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Audit Entries"),
        description: t("Monitor and review system activity across your organization"),
      }}
    >
      <div className="flex flex-col gap-2 p-4">
        <AuditAlert />
        <DataTableLazyComponent>
          <AuditLogTable />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
