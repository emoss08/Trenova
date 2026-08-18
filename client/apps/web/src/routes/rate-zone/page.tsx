import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/rate-zone-table"));

export function RateZonePage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Rate Zones"
        description="Name a market area once and price against it, instead of listing every postal prefix it covers"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
