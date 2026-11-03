import type { MapShipment } from "./map-shipment-model";
import type { Latlng } from "./shipment-route-coordinates";

export type MapLane = {
  key: string;
  origin: Latlng;
  destination: Latlng;
  originLabel: string;
  destinationLabel: string;
  shipmentIds: string[];
  lateCount: number;
};

/**
 * One lane per origin and destination place, carrying every shipment that runs
 * it. Lightest first, so the busiest lanes draw on top.
 */
export function groupLanes(models: readonly MapShipment[]): MapLane[] {
  const lanes = new Map<string, MapLane>();
  for (const model of models) {
    if (!model.lane) continue;
    let lane = lanes.get(model.lane.key);
    if (!lane) {
      lane = { ...model.lane, shipmentIds: [], lateCount: 0 };
      lanes.set(model.lane.key, lane);
    }
    lane.shipmentIds.push(model.id);
    if (model.stage === "Late") lane.lateCount += 1;
  }
  return [...lanes.values()].sort((a, b) => a.shipmentIds.length - b.shipmentIds.length);
}

/** A lane's stroke weight: thicker for more loads, capped so a busy lane never hides the map. */
export function laneWeight(shipmentCount: number): number {
  return Math.min(2 + (shipmentCount - 1) * 1.5, 10);
}

/**
 * A gentle arc from origin to destination, bowed to the left of travel, so a
 * lane and its return lane draw apart instead of on top of each other.
 * Computed in a plane with longitude scaled by the cosine of the mid latitude,
 * which keeps the bow even at any latitude the map shows.
 */
export function laneArcPath(
  origin: Latlng,
  destination: Latlng,
  bend = 0.16,
  steps = 24,
): Latlng[] {
  const scale = Math.cos((((origin.lat + destination.lat) / 2) * Math.PI) / 180) || 1;
  const x0 = origin.lng * scale;
  const y0 = origin.lat;
  const x2 = destination.lng * scale;
  const y2 = destination.lat;
  const cx = (x0 + x2) / 2 - (y2 - y0) * bend;
  const cy = (y0 + y2) / 2 + (x2 - x0) * bend;

  const path: Latlng[] = [];
  for (let i = 0; i <= steps; i++) {
    const t = i / steps;
    const u = 1 - t;
    const x = u * u * x0 + 2 * u * t * cx + t * t * x2;
    const y = u * u * y0 + 2 * u * t * cy + t * t * y2;
    path.push({ lat: y, lng: x / scale });
  }
  return path;
}
