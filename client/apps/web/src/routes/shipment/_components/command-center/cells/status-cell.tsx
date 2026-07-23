import { ShipmentStatusBadge } from "@/components/status-badge";
import type { Shipment } from "@/types/shipment";

export function StatusCell({ shipment }: { shipment: Shipment }) {
  return (
    <div className="inline-flex flex-col items-start gap-1">
      <ShipmentStatusBadge status={shipment.status} />
    </div>
  );
}
