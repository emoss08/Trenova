import { DataTableLazyComponent, LazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { ShipmentMapPanelBoundary } from "./_components/map/shipment-map-panel";

const Table = lazy(() => import("./_components/shipment-table"));
const ShipmentAnalytics = lazy(() => import("./_components/analytics/shipment-analytics"));
const ShipmentMapPanel = lazy(() => import("./_components/map/shipment-map-panel"));

export function ShipmentsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Shipments",
        description: "Manage and configure shipments for your organization",
      }}
    >
      <LazyComponent>
        <ShipmentAnalytics />
      </LazyComponent>
      <ShipmentMapPanelBoundary>
        <ShipmentMapPanel />
      </ShipmentMapPanelBoundary>
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
