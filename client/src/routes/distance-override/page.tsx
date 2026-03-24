import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/distance-override-table"));

export function DistanceOverridesPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Distance Overrides"
        description="Override calculated distances between location pairs for routing and billing adjustments"
      />
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </AdminPageLayout>
  );
}











