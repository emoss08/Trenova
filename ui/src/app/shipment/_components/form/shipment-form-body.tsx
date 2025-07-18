import { LazyComponent } from "@/components/error-boundary";
import { ScrollArea, ScrollAreaShadow } from "@/components/ui/scroll-area";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { lazy } from "react";

const ShipmentDetailsHeader = lazy(() => import("./shipment-details-header"));

export function ShipmentFormContent({
  children,
  selectedShipment,
}: {
  children: React.ReactNode;
  selectedShipment?: ShipmentSchema | null;
}) {
  return (
    <ShipmentScrollAreaOuter>
      <LazyComponent>
        <ShipmentDetailsHeader selectedShipment={selectedShipment} />
      </LazyComponent>
      <ScrollArea className="flex flex-col overflow-y-auto max-h-[calc(100vh-8.5rem)]">
        <ShipmentScrollAreaInner>{children}</ShipmentScrollAreaInner>
        <ScrollAreaShadow />
      </ScrollArea>
    </ShipmentScrollAreaOuter>
  );
}

function ShipmentScrollAreaInner({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4 p-4 pb-16">{children}</div>;
}

function ShipmentScrollAreaOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col">{children}</div>;
}
