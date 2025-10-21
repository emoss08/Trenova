import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentDetailsHeader } from "./shipment-details-header";

export function ShipmentFormContent({
  children,
  selectedShipment,
}: {
  children: React.ReactNode;
  selectedShipment?: ShipmentSchema | null;
}) {
  return (
    <>
      <ShipmentDetailsHeader selectedShipment={selectedShipment} />
      {children}
    </>
  );
}
