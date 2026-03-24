import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/location-table"));

export function LocationsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Locations",
        description: "Manage and configure locations for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
