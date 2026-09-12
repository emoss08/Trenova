import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/custom-field-definition-table"));

export function CustomFieldDefinitionsPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Custom Field Definitions")}
        description={t("Define custom fields for trailers, workers, and other resources")}
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
