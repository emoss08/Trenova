import {
  shipmentStatusSchema,
  stopTypeSchema,
  type Shipment,
  type ShipmentStatus,
  type Stop,
} from "@/types/shipment";

function findFirstStop(shipment: Shipment, type: Stop["type"]) {
  if (!shipment.moves?.length) return null;

  for (const move of shipment.moves) {
    if (!move.stops?.length) continue;
    for (const stop of move.stops) {
      if (stop.type === type) return stop;
    }
  }

  return null;
}

function findLastStop(shipment: Shipment, type: Stop["type"]) {
  if (!shipment.moves?.length) return null;

  const { moves } = shipment;
  for (let i = moves.length - 1; i >= 0; i--) {
    const stops = moves[i].stops;
    if (!stops?.length) continue;
    for (let j = stops.length - 1; j >= 0; j--) {
      if (stops[j].type === type) return stops[j];
    }
  }

  return null;
}

export function getOriginStop(shipment: Shipment) {
  return findFirstStop(shipment, stopTypeSchema.enum.Pickup);
}

export function getDestinationStop(shipment: Shipment) {
  return findLastStop(shipment, stopTypeSchema.enum.Delivery);
}

export function getOriginLocation(shipment: Shipment) {
  return getOriginStop(shipment)?.location ?? null;
}

export function getDestinationLocation(shipment: Shipment) {
  return getDestinationStop(shipment)?.location ?? null;
}

export function getOrderedStops(shipment: Shipment): Stop[] {
  if (!shipment.moves?.length) return [];
  const flat: Stop[] = [];
  for (const move of shipment.moves) {
    if (!move.stops?.length) continue;
    for (const stop of move.stops) flat.push(stop);
  }
  return flat;
}

export function getTotalMiles(shipment: Shipment) {
  if (!shipment.moves?.length) return 0;

  return shipment.moves.reduce((total, move) => total + (move.distance ?? 0), 0);
}

type ShipmentProgressVariant = "default" | "success" | "warning" | "error";

const SHIPMENT_STATUS_PROGRESS: Record<
  ShipmentStatus,
  { value: number; variant: ShipmentProgressVariant }
> = {
  [shipmentStatusSchema.enum.New]: { value: 5, variant: "warning" },
  [shipmentStatusSchema.enum.PartiallyAssigned]: { value: 15, variant: "warning" },
  [shipmentStatusSchema.enum.Assigned]: { value: 25, variant: "warning" },
  [shipmentStatusSchema.enum.InTransit]: { value: 50, variant: "warning" },
  [shipmentStatusSchema.enum.Delayed]: { value: 50, variant: "warning" },
  [shipmentStatusSchema.enum.PartiallyCompleted]: { value: 75, variant: "warning" },
  [shipmentStatusSchema.enum.Completed]: { value: 90, variant: "success" },
  [shipmentStatusSchema.enum.ReadyToInvoice]: { value: 95, variant: "success" },
  [shipmentStatusSchema.enum.Invoiced]: { value: 100, variant: "success" },
  [shipmentStatusSchema.enum.Canceled]: { value: 100, variant: "error" },
};

export function getShipmentProgress(status: ShipmentStatus) {
  return SHIPMENT_STATUS_PROGRESS[status];
}

export type ShipmentEtaTone = "ontime" | "watch" | "late" | "delivered" | "pending";

export function getShipmentEtaTone(shipment: Shipment): ShipmentEtaTone {
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

export function isEligibleTenderStatus(tenderStatus: Shipment["tenderStatus"]) {
  return (
    !tenderStatus ||
    tenderStatus === "Rejected" ||
    tenderStatus === "Expired" ||
    tenderStatus === "Canceled"
  );
}
