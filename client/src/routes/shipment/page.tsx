import { DataTableLazyComponent, LazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { RightStack } from "./_components/command-center/right-stack";
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
      <div className="cc-workspace flex flex-col gap-3">
        <LazyComponent>
          <ShipmentAnalytics />
        </LazyComponent>
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-[minmax(0,1fr)_minmax(320px,380px)]">
          <ShipmentMapPanelBoundary>
            <ShipmentMapPanel />
          </ShipmentMapPanelBoundary>
          {/*
            Pin the right-stack column to the same clamp() height as the
            map so the grid row doesn't stretch when content is tall. The
            aside inside uses flex-1 / min-h-0 to keep its modules within
            this box. We deliberately allow visible overflow so the
            floating "Add panel" chip (positioned at -top-7) can sit above
            the stack the way the design specifies.
          */}
          <div className="relative h-[clamp(420px,calc(100vh-380px),540px)] min-h-0">
            <LazyComponent>
              <RightStack />
            </LazyComponent>
          </div>
        </div>
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
