import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/distance-profile-table"));

export function DistanceProfilesPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Distance Profiles"
        description="Manage business-unit routing policy used by distance calculations"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
