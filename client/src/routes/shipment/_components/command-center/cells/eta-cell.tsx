import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import {
  getDestinationStop,
  getShipmentEtaTone,
  type ShipmentEtaTone,
} from "@/lib/shipment-utils";
import type { Shipment } from "@/types/shipment";

const TONE_CLASS: Record<ShipmentEtaTone, string> = {
  ontime: "text-foreground",
  watch: "text-warning",
  late: "text-destructive",
  delivered: "text-success",
  pending: "text-muted-foreground",
};

export function EtaCell({ shipment }: { shipment: Shipment }) {
  const stop = getDestinationStop(shipment);
  const etaTimestamp = stop?.scheduledWindowEnd ?? stop?.scheduledWindowStart ?? null;
  const tone = getShipmentEtaTone(shipment);

  if (!etaTimestamp) {
    return <span className="font-table text-[11.5px] text-muted-foreground">—</span>;
  }

  const eta = formatToUserTimezone(etaTimestamp, {
    showTimeZone: false,
    showSeconds: false,
  });

  return (
    <div className="flex flex-col gap-0.5">
      <span
        className={cn(
          "font-table text-[11.5px] font-medium tabular-nums",
          TONE_CLASS[tone],
        )}
      >
        {eta}
      </span>
    </div>
  );
}
