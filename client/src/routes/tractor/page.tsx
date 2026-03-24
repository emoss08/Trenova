import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/tractor-table"));

export function TractorsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Tractors",
        description: "Manage and configure tractors for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
