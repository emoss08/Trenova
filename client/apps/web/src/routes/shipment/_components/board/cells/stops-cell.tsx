import { StopStripCell } from "@/components/data-table/cells/stop-strip-cell";
import { getStopStrip } from "@/lib/shipment-utils";
import { useNow } from "@trenova/shared/hooks/use-now";
import type { Shipment } from "@trenova/shared/types/shipment";

export function StopsCell({ shipment }: { shipment: Shipment }) {
  const now = useNow("minute");
  return <StopStripCell stops={getStopStrip(shipment, now)} now={now} />;
}
