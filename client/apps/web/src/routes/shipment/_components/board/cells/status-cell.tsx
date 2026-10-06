import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";

export function StatusCell({ shipment }: { shipment: Shipment }) {
  const { operationType } = useShipmentCapabilities();
  return <ShipmentStatusBadge status={shipment.status} brokered={operationType === "brokerage"} />;
}
