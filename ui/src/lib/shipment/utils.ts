import { Shipment } from "@/types/shipment";
import { useMemo } from "react";
import { LocationSchema } from "../schemas/location-schema";
import type { ShipmentSchema } from "../schemas/shipment-schema";

const STOP_TYPES = {
  PICKUP: "Pickup",
  DELIVERY: "Delivery",
} as const;

export function getOriginStopInfo(shipment: ShipmentSchema) {
  if (!shipment.moves?.length) {
    return null;
  }

  const firstMove = shipment.moves[0];
  if (!firstMove.stops?.length) {
    return null;
  }

  for (const stop of firstMove.stops) {
    if (stop.type === STOP_TYPES.PICKUP) {
      return stop;
    }
  }

  return null;
}

function calculateOriginLocation(shipment: ShipmentSchema) {
  const originStop = getOriginStopInfo(shipment);

  if (!originStop) {
    return null;
  }

  if (!originStop.location) {
    return null;
  }

  return originStop.location;
}

export function getDestinationStopInfo(shipment: ShipmentSchema) {
  if (!shipment.moves?.length) {
    return null;
  }

  const { moves } = shipment;
  for (let i = moves.length - 1; i >= 0; i--) {
    const move = moves[i];
    if (!move.stops?.length) continue;

    for (let j = move.stops.length - 1; j >= 0; j--) {
      const stop = move.stops[j];
      if (stop.type === STOP_TYPES.DELIVERY) {
        return stop;
      }
    }
  }
}

function calculateDestinationLocation(shipment: ShipmentSchema) {
  const destinationStop = getDestinationStopInfo(shipment);

  if (!destinationStop) {
    return null;
  }

  if (!destinationStop.location) {
    return null;
  }

  return destinationStop.location;
}

const locationCache = new WeakMap<
  ShipmentSchema,
  {
    origin: LocationSchema | null;
    destination: LocationSchema | null;
  }
>();

function useShipmentLocations(shipment?: ShipmentSchema) {
  return useMemo(
    () => ({
      origin: shipment ? calculateOriginLocation(shipment) : null,
      destination: shipment ? calculateDestinationLocation(shipment) : null,
    }),
    [shipment],
  ); // React's useMemo is enough for component-level caching
}

export const ShipmentLocations = {
  useLocations: useShipmentLocations,
  getOrigin: (shipment: ShipmentSchema) => {
    const cached = locationCache.get(shipment);
    if (cached) return cached.origin;
    const result = calculateOriginLocation(shipment);
    locationCache.set(shipment, {
      origin: result,
      destination: calculateDestinationLocation(shipment),
    });
    return result;
  },

  getDestination: (shipment: ShipmentSchema) => {
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

export function calculateShipmentMileage(shipment: ShipmentSchema) {
  // First find all of the moves for the shipment
  const { moves } = shipment;
  if (!moves?.length) {
    return 0;
  }

  // Second, loop through all of the moves and sum up the distance for each move

  let totalDistance = 0;
  for (const move of moves) {
    const { distance } = move;
    if (!distance) {
      continue;
    }

    if (typeof distance !== "number") {
      throw new Error("Distance is not a number");
    }

    totalDistance += distance;
  }

  return totalDistance;
}

export function calculateShipmentDuration(shipment: ShipmentSchema) {
  const originStop = getOriginStopInfo(shipment);
  const destinationStop = getDestinationStopInfo(shipment);

  if (
    !originStop ||
    typeof originStop.plannedArrival !== "number" ||
    !destinationStop ||
    typeof destinationStop.plannedArrival !== "number"
  ) {
    return 0;
  }

  const duration = destinationStop.plannedArrival - originStop.plannedArrival;

  // Return 0 if duration is negative, otherwise return the duration
  return duration > 0 ? duration : 0;
}

export function getShipmentStopCount(shipment: ShipmentSchema) {
  // First find all of the moves for the shipment
  const { moves } = shipment;
  if (!moves?.length) {
    return 0;
  }

  // Second, loop through all of the moves and sum up the stop count
  let totalStopCount = 0;
  for (const move of moves) {
    const { stops } = move;
    if (!stops?.length) continue;

    totalStopCount += stops.length;
  }

  return totalStopCount;
}
