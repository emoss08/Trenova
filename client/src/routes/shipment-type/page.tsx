import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/shipment-type-table"));

export function ShipmentTypesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Shipment Types",
        description: "Manage and configure shipment types for your organization",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
