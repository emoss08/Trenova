import { stopTypeSchema, type Shipment, type Stop } from "@/types/shipment";

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
