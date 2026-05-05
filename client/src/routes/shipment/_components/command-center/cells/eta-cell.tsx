import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { getDestinationStop } from "@/lib/shipment-utils";
import type { Shipment } from "@/types/shipment";

type EtaTone = "ontime" | "watch" | "late" | "delivered" | "pending";

function deriveEtaTone(shipment: Shipment): EtaTone {
  switch (shipment.status) {
    case "Completed":
    case "Invoiced":
    case "ReadyToInvoice":
      return "delivered";
    case "Delayed":
      return "late";
    case "InTransit":
    case "PartiallyCompleted": {
      // If we're past the destination's scheduled-window midpoint we treat it
      // as "watch" — running close to or past the booked time.
      const stop = getDestinationStop(shipment);
      if (!stop?.scheduledWindowStart) return "ontime";
      const end = stop.scheduledWindowEnd ?? stop.scheduledWindowStart;
      const mid = (stop.scheduledWindowStart + end) / 2;
      const nowSeconds = Math.floor(Date.now() / 1000);
      if (nowSeconds >= end) return "late";
      if (nowSeconds >= mid) return "watch";
      return "ontime";
    }
    case "Canceled":
      return "pending";
    default:
      return "pending";
  }
}

const TONE_CLASS: Record<EtaTone, string> = {
  ontime: "text-foreground",
  watch: "text-warning",
  late: "text-destructive",
  delivered: "text-success",
  pending: "text-muted-foreground",
};

export function EtaCell({ shipment }: { shipment: Shipment }) {
  const stop = getDestinationStop(shipment);
  const etaTimestamp = stop?.scheduledWindowEnd ?? stop?.scheduledWindowStart ?? null;
  const tone = deriveEtaTone(shipment);

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
