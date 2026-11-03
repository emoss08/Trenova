import type { WeatherAlertCategory, WeatherAlertFeature } from "@/types/weather-alert";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";

/* Fixtures follow the ShipmentFields GraphQL fragment and the weather alert
   GeoJSON contract (types/weather-alert.ts), not what the map code reads. */

export const PLACES = {
  dallas: { city: "Dallas", st: "TX", lat: 32.78, lng: -96.8 },
  memphis: { city: "Memphis", st: "TN", lat: 35.15, lng: -90.05 },
  atlanta: { city: "Atlanta", st: "GA", lat: 33.75, lng: -84.39 },
  houston: { city: "Houston", st: "TX", lat: 29.76, lng: -95.37 },
  chicago: { city: "Chicago", st: "IL", lat: 41.88, lng: -87.63 },
} as const;

type Place = (typeof PLACES)[keyof typeof PLACES];

export type StopSpec = {
  at: Place | null;
  type?: Stop["type"];
  status?: Stop["status"];
  window?: number;
};

let stopSeq = 0;

export function stop({ at, type = "Pickup", status = "New", window = 0 }: StopSpec): Stop {
  stopSeq += 1;
  return {
    id: `stp_${stopSeq}`,
    locationId: `loc_${stopSeq}`,
    type,
    status,
    scheduleType: "Open",
    sequence: stopSeq,
    scheduledWindowStart: window,
    scheduledWindowEnd: null,
    actualArrival: null,
    actualDeparture: null,
    location: at
      ? {
          id: `loc_${stopSeq}`,
          name: at.city,
          city: at.city,
          latitude: at.lat,
          longitude: at.lng,
          state: { id: `st_${at.st}`, name: at.st, abbreviation: at.st },
        }
      : null,
  } as unknown as Stop;
}

export function shipment(
  id: string | undefined,
  stops: StopSpec[],
  extra: Partial<Shipment> & Record<string, unknown> = {},
): Shipment {
  return {
    id,
    proNumber: id ? `PRO-${id}` : undefined,
    status: "InTransit",
    stage: "Moving",
    eta: null,
    customer: { id: "cus_1", name: "Halverson Foods", code: "HALV" },
    moves: [{ id: `mov_${id}`, stops: stops.map(stop), assignment: null, carrierAssignment: null }],
    ...extra,
  } as unknown as Shipment;
}

/** A closed square ring of `half` degrees around a point, as GeoJSON [lng, lat] pairs. */
export function square(lat: number, lng: number, half: number): number[][] {
  return [
    [lng - half, lat - half],
    [lng + half, lat - half],
    [lng + half, lat + half],
    [lng - half, lat + half],
    [lng - half, lat - half],
  ];
}

export function alert(
  id: string,
  rings: number[][][],
  {
    severity = "Severe",
    event = "Flood Warning",
    category = "flood_water",
    multi = false,
  }: {
    severity?: string;
    event?: string;
    category?: WeatherAlertCategory;
    multi?: boolean;
  } = {},
): WeatherAlertFeature {
  return {
    type: "Feature",
    id,
    geometry: multi
      ? { type: "MultiPolygon", coordinates: rings.map((ring) => [ring]) }
      : { type: "Polygon", coordinates: [rings[0]] },
    properties: {
      id,
      nwsId: `urn:oid:${id}`,
      event,
      severity,
      alertCategory: category,
      areaDesc: "Union and Ouachita counties",
      firstSeenAt: 1,
      lastUpdatedAt: 1,
    },
  };
}
