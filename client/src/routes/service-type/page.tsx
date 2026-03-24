import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/service-type-table"));

export function ServiceTypesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Service Types",
        description: "Manage and configure service types for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
