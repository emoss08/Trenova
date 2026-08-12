import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/carrier-table"));

export function CarriersPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Carriers",
        description: "Manage and configure carriers for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
