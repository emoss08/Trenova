import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/hold-reason-table"));

export function HoldReasonsPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Hold Reasons")}
        description={t("Manage and configure hold reasons for your organization")}
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
