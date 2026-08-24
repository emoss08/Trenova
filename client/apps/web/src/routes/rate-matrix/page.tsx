import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/rate-matrix-table"));

export function RateMatrixPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Rate Matrices"
        description="Price a tariff the way it was published — a grid of zones, weight breaks and classes — instead of one lane per cell"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
