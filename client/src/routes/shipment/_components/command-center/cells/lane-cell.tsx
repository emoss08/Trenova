import {
  getDestinationLocation,
  getOriginLocation,
  getShipmentProgress,
  getTotalMiles,
} from "@/lib/shipment-utils";
import { cn } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";
import { ArrowRight } from "lucide-react";
import { useCommandCenterStore } from "../store";

type LaneToneClass =
  | "lane-bar-default"
  | "lane-bar-success"
  | "lane-bar-warning"
  | "lane-bar-danger";

function toneClass(status: Shipment["status"]): LaneToneClass {
  switch (status) {
    case "Completed":
    case "Invoiced":
    case "ReadyToInvoice":
      return "lane-bar-success";
    case "Delayed":
      return "lane-bar-danger";
    case "Canceled":
      return "lane-bar-warning";
    default:
      return "lane-bar-default";
  }
}

export function LaneCell({ shipment }: { shipment: Shipment }) {
  const highlightId = useCommandCenterStore.use.highlightId();
  const isHighlighted = !!shipment.id && highlightId === shipment.id;

  const originLocation = getOriginLocation(shipment);
  const destinationLocation = getDestinationLocation(shipment);
  const originCode = originLocation?.code ?? "—";
  const destinationCode = destinationLocation?.code ?? "—";
  const miles = getTotalMiles(shipment);
  const progress = getShipmentProgress(shipment.status);
  const commodityName = shipment.commodities?.[0]?.commodity?.name ?? null;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex flex-row items-center gap-1.5">
        <span
          className={cn(
            "font-table text-[11.5px] font-semibold tabular-nums",
            isHighlighted && "text-brand",
          )}
        >
          {originCode}
        </span>
        <ArrowRight className="size-3 shrink-0 text-muted-foreground" />
        <span className="truncate font-table text-[11.5px] font-semibold tabular-nums">
          {destinationCode}
        </span>
        <div className={cn("lane-bar ml-1 max-w-20 flex-1", toneClass(shipment.status))}>
          <span style={{ width: `${progress.value}%` }} />
        </div>
        <span className="font-table text-[9.5px] text-muted-foreground tabular-nums">
          {progress.value}%
        </span>
      </div>
      <div className="font-table text-[9.5px] text-muted-foreground tabular-nums">
        {miles}mi{commodityName ? ` · ${commodityName}` : ""}
      </div>
    </div>
  );
}
