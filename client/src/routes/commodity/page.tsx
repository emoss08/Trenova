import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/commodity-table"));

export function CommoditiesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Commodities",
        description: "Manage and configure commodities for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
