import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/service-failure-reason-code-table"));

export function ServiceFailureReasonCodesPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Service Failure Reason Codes")}
        description={t("Manage operational exception reasons and EDI 214 defaults")}
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
