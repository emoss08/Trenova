import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ShipmentBillingQueueBadge } from "../../shipment-billing-queue-status";

export function StatusCell({ shipment }: { shipment: Shipment }) {
  return (
    <div className="inline-flex flex-col items-start gap-1">
      <ShipmentStatusBadge status={shipment.status} />
      <ShipmentBillingQueueBadge status={shipment.billingTransferStatus} />
    </div>
  );
}
