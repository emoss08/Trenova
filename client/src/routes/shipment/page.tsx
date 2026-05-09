import { DataTableLazyComponent, LazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { ShipmentMapPanelBoundary } from "./_components/map/shipment-map-panel";

const Table = lazy(() => import("./_components/shipment-table"));
const ShipmentAnalytics = lazy(() => import("./_components/analytics/kpi/kpi-rail"));
const ShipmentMapPanel = lazy(() => import("./_components/map/shipment-map-panel"));
const RightStack = lazy(() => import("./_components/command-center/right-stack"));
const BottomModules = lazy(
  () => import("./_components/command-center/bottom-modules"),
);

export function ShipmentsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Shipments",
        description: "Manage and configure shipments for your organization",
      }}
    >
      <div className="cc-workspace flex flex-col gap-3">
        <LazyComponent>
          <ShipmentAnalytics />
        </LazyComponent>
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-[minmax(0,1fr)_minmax(320px,380px)]">
          <ShipmentMapPanelBoundary>
            <ShipmentMapPanel />
          </ShipmentMapPanelBoundary>
          <div className="relative h-[clamp(420px,calc(100vh-380px),540px)] min-h-0">
            <LazyComponent>
              <RightStack />
            </LazyComponent>
          </div>
        </div>
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
        <LazyComponent>
          <BottomModules />
        </LazyComponent>
      </div>
    </PageLayout>
  );
}
