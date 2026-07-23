import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/stored-mileage-table"));

export function StoredMileagesPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Stored Mileages"
        description="Review reusable mileage records captured from PC*Miler calculations"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
