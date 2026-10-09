import { ArrivalWindowCell } from "@/components/data-table/cells/arrival-window-cell";
import { getDestinationStop } from "@/lib/shipment-utils";
import type { Shipment } from "@trenova/shared/types/shipment";

/** The delivery against its window: where the truck landed, or where it is expected to. */
export function ArrivalCell({ shipment }: { shipment: Shipment }) {
  const stop = getDestinationStop(shipment);
  const arrived = stop?.actualArrival ?? null;
  return (
    <ArrivalWindowCell
      arrival={arrived ?? shipment.eta?.estimatedArrival}
      arrived={arrived !== null}
      windowStart={stop?.scheduledWindowStart}
      windowEnd={stop?.scheduledWindowEnd}
    />
  );
}
