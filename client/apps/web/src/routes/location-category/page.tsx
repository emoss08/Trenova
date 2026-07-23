import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/location-category-table"));

export function LocationCategoriesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Location Categories",
        description: "Manage and configure location categories for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
