import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/custom-field-definition-table"));

export function CustomFieldDefinitionsPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Custom Field Definitions"
        description="Define custom fields for trailers, workers, and other resources"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
