import { resolveCoverage } from "@/lib/shipment-board/coverage";
import { ACTIVE_STAGES, isActiveStage, type ActiveStage } from "@/lib/shipment-board/stage";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import {
  getShipmentStopsWithCoords,
  pickCurrentStopIndex,
  readLatLng,
  type Latlng,
  type StopWithCoord,
} from "./shipment-route-coordinates";

/** The stages the live map draws, in the board's order: what needs a hand first. */
export const MAP_STAGES = ACTIVE_STAGES;
export type MapStage = ActiveStage;

/** A delivered load needs nothing from a dispatcher, so it starts hidden. */
export const DEFAULT_HIDDEN_STAGES: readonly MapStage[] = ["Delivered"];

export const isMapStage = isActiveStage;

export type MapLaneEnds = {
  key: string;
  origin: Latlng;
  destination: Latlng;
  originLabel: string;
  destinationLabel: string;
};

/** One shipment as the live map draws it: one mark, its stops, and its lane. */
export type MapShipment = {
  id: string;
  shipment: Shipment;
  stage: MapStage;
  /** Stops that have coordinates, in route order. */
  stops: StopWithCoord[];
  /** The stop the load is at or heading to; its mark sits there. */
  currentIndex: number;
  position: Latlng;
  lane: MapLaneEnds | null;
  searchText: string;
};

const CLOSED_STOP_STATUSES = new Set<Stop["status"]>(["Completed", "Canceled"]);

/** "Dallas, TX" for a stop's location, or null when it has no city. */
export function placeLabel(location: Stop["location"] | null | undefined): string | null {
  if (!location?.city) return null;
  return [location.city, location.state?.abbreviation].filter(Boolean).join(", ");
}

function laneEnds(shipment: Shipment): MapLaneEnds | null {
  const originStop = getOriginStop(shipment);
  const destinationStop = getDestinationStop(shipment);
  const origin = readLatLng(originStop?.location ?? null);
  const destination = readLatLng(destinationStop?.location ?? null);
  const originLabel = placeLabel(originStop?.location);
  const destinationLabel = placeLabel(destinationStop?.location);
  if (!origin || !destination || !originLabel || !destinationLabel) return null;
  return {
    key: [originLabel, destinationLabel].join(" > "),
    origin,
    destination,
    originLabel,
    destinationLabel,
  };
}

function searchTextOf(shipment: Shipment, stops: StopWithCoord[]): string {
  const coverage = resolveCoverage(shipment);
  return [
    shipment.proNumber,
    shipment.bol,
    shipment.customer?.name,
    shipment.customer?.code,
    "name" in coverage ? coverage.name : null,
    "detail" in coverage ? coverage.detail : null,
    ...stops.map(({ stop }) => placeLabel(stop.location)),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

/**
 * The shipments the map can place. A load with no stage yet is treated as
 * needing coverage, as the board's lane cell does; a canceled one, or one with
 * no stop the map can locate, is left off.
 */
export function buildMapShipments(shipments: readonly Shipment[]): MapShipment[] {
  const result: MapShipment[] = [];
  for (const shipment of shipments) {
    if (!shipment.id) continue;
    const stage = shipment.stage ?? "NeedsCoverage";
    if (!isMapStage(stage)) continue;
    const stops = getShipmentStopsWithCoords(shipment);
    if (stops.length === 0) continue;
    const currentIndex = pickCurrentStopIndex(stops);
    result.push({
      id: shipment.id,
      shipment,
      stage,
      stops,
      currentIndex,
      position: stops[currentIndex].latlng,
      lane: laneEnds(shipment),
      searchText: searchTextOf(shipment, stops),
    });
  }
  return result;
}

/** The first stop still to be worked, or null once every stop is closed. */
export function nextOpenStop(model: MapShipment): StopWithCoord | null {
  return model.stops.find(({ stop }) => !CLOSED_STOP_STATUSES.has(stop.status)) ?? null;
}

/**
 * The route still ahead of the load: from the last stop it left through every
 * stop it has yet to reach. A delivered load has none.
 */
export function remainingPath(model: MapShipment): Latlng[] {
  if (model.stage === "Delivered") return [];
  const firstOpen = model.stops.findIndex(({ stop }) => !CLOSED_STOP_STATUSES.has(stop.status));
  if (firstOpen === -1) return [];
  return model.stops.slice(Math.max(0, firstOpen - 1)).map(({ latlng }) => latlng);
}

/** Whether the leg from stop `index` to the next has been driven. */
export function isLegDriven(model: MapShipment, index: number): boolean {
  return model.stops[index + 1]?.stop.status === "Completed";
}

export function matchesMapSearch(model: MapShipment, query: string): boolean {
  const needle = query.trim().toLowerCase();
  return !needle || model.searchText.includes(needle);
}

const NULLS_LAST = Number.POSITIVE_INFINITY;

function firstWindowStart(model: MapShipment): number {
  return model.stops[0]?.stop.scheduledWindowStart ?? NULLS_LAST;
}

/* Within a stage the rail leads with what is most urgent: the latest load, the
   next arrival, the next pickup, the most recent delivery. */
const STAGE_ORDER: Record<MapStage, (a: MapShipment, b: MapShipment) => number> = {
  Late: (a, b) =>
    (a.shipment.eta?.slackMinutes ?? NULLS_LAST) - (b.shipment.eta?.slackMinutes ?? NULLS_LAST),
  Moving: (a, b) =>
    (a.shipment.eta?.estimatedArrival ?? NULLS_LAST) -
    (b.shipment.eta?.estimatedArrival ?? NULLS_LAST),
  NeedsCoverage: (a, b) => firstWindowStart(a) - firstWindowStart(b),
  Scheduled: (a, b) => firstWindowStart(a) - firstWindowStart(b),
  Delivered: (a, b) => (b.shipment.actualDeliveryDate ?? 0) - (a.shipment.actualDeliveryDate ?? 0),
};

export type MapStageGroup = { stage: MapStage; items: MapShipment[] };

/** The shipments grouped by stage, in MAP_STAGES order, each group most urgent first. */
export function groupByStage(models: readonly MapShipment[]): MapStageGroup[] {
  return MAP_STAGES.map((stage) => ({
    stage,
    items: models.filter((model) => model.stage === stage).sort(STAGE_ORDER[stage]),
  })).filter((group) => group.items.length > 0);
}

/** How many shipments sit in each stage, counting every stage, even an empty one. */
export function countByStage(models: readonly MapShipment[]): Record<MapStage, number> {
  const counts = Object.fromEntries(MAP_STAGES.map((stage) => [stage, 0])) as Record<
    MapStage,
    number
  >;
  for (const model of models) counts[model.stage] += 1;
  return counts;
}
