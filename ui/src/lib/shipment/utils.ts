import { Shipment } from "@/types/shipment";
import { useMemo } from "react";
import { LocationSchema } from "../schemas/location-schema";

const STOP_TYPES = {
  PICKUP: "Pickup",
  DELIVERY: "Delivery",
} as const;

function calculateOriginLocation(shipment: Shipment) {
  if (!shipment.moves?.length) {
    return null;
  }

  const firstMove = shipment.moves[0];
  if (!firstMove.stops?.length) {
    return null;
  }

  for (const stop of firstMove.stops) {
    if (stop.type === STOP_TYPES.PICKUP && stop.location) {
      return stop.location;
    }
  }

  return null;
}

function calculateDestinationLocation(shipment: Shipment) {
  if (!shipment.moves?.length) {
    return null;
  }

  const { moves } = shipment;
  for (let i = moves.length - 1; i >= 0; i--) {
    const move = moves[i];
    if (!move.stops?.length) continue;

    for (let j = move.stops.length - 1; j >= 0; j--) {
      const stop = move.stops[j];
      if (stop.type === STOP_TYPES.DELIVERY && stop.location) {
        return stop.location;
      }
    }
  }

  return null;
}

const locationCache = new WeakMap<
  Shipment,
  {
    origin: LocationSchema | null;
    destination: LocationSchema | null;
  }
>();

function useShipmentLocations(shipment: Shipment) {
  return useMemo(
    () => ({
      origin: calculateOriginLocation(shipment),
      destination: calculateDestinationLocation(shipment),
    }),
    [shipment],
  ); // React's useMemo is enough for component-level caching
}

export const ShipmentLocations = {
  useLocations: useShipmentLocations,
  getOrigin: (shipment: Shipment) => {
    const cached = locationCache.get(shipment);
    if (cached) return cached.origin;
    const result = calculateOriginLocation(shipment);
    locationCache.set(shipment, {
      origin: result,
      destination: calculateDestinationLocation(shipment),
    });
    return result;
  },

  getDestination: (shipment: Shipment) => {
    const cached = locationCache.get(shipment);
    if (cached) return cached.destination;
    const result = calculateDestinationLocation(shipment);
    locationCache.set(shipment, {
      origin: calculateOriginLocation(shipment),
      destination: result,
    });
    return result;
  },
  invalidate: (shipment: Shipment) => {
    locationCache.delete(shipment);
  },
} as const;
