import { ShipmentStatusBadge } from "@/components/status-badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { toTitleCase } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";

export function StatusCell({ shipment }: { shipment: Shipment }) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div className="inline-flex">
            <ShipmentStatusBadge status={shipment.status} />
          </div>
        }
      />
      <TooltipContent side="top">Status: {toTitleCase(shipment.status)}</TooltipContent>
    </Tooltip>
  );
}
