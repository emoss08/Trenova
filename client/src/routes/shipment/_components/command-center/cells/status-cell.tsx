import { ShipmentStatusBadge } from "@/components/status-badge";
import type { Shipment } from "@/types/shipment";

export function StatusCell({ shipment }: { shipment: Shipment }) {
  return (
    <div className="inline-flex">
      <ShipmentStatusBadge status={shipment.status} />
    </div>
  );
}
