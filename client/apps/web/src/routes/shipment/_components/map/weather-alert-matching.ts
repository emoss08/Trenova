import type { WeatherAlertFeature } from "@/types/weather-alert";
import { remainingPath, type MapShipment } from "./map-shipment-model";
import type { Latlng } from "./shipment-route-coordinates";

/**
 * Where an alert meets a shipment: the stop the load is at or heading to sits
 * inside it, or the route still ahead passes through it.
 */
export type AlertHit = { alertId: string; shipmentId: string; at: "stop" | "route" };

export type AlertMatches = {
  byShipment: Map<string, AlertHit[]>;
  byAlert: Map<string, AlertHit[]>;
};

type Ring = { points: number[][]; south: number; north: number; west: number; east: number };

/** Degrees between route samples; about seven miles, finer than a county. */
const SAMPLE_STEP_DEGREES = 0.1;
const MAX_SAMPLES_PER_LEG = 400;

const SEVERITY_RANK: Record<string, number> = { Extreme: 0, Severe: 1, Moderate: 2, Minor: 3 };

export function severityRank(severity: string | null | undefined): number {
  return SEVERITY_RANK[severity ?? ""] ?? 4;
}

/** The outer rings of an alert's Polygon or MultiPolygon, as [lng, lat] pairs. */
export function alertOuterRings(feature: WeatherAlertFeature): number[][][] {
  const geometry = feature.geometry;
  if (!geometry) return [];
  if (geometry.type === "Polygon") {
    const outer = (geometry.coordinates as number[][][])[0];
    return outer ? [outer] : [];
  }
  if (geometry.type === "MultiPolygon") {
    return (geometry.coordinates as number[][][][])
      .map((polygon) => polygon[0])
      .filter((ring): ring is number[][] => !!ring && ring.length > 0);
  }
  return [];
}

function toRing(points: number[][]): Ring {
  let south = 90;
  let north = -90;
  let west = 180;
  let east = -180;
  for (const [lng, lat] of points) {
    south = Math.min(south, lat);
    north = Math.max(north, lat);
    west = Math.min(west, lng);
    east = Math.max(east, lng);
  }
  return { points, south, north, west, east };
}

function insideRing({ lat, lng }: Latlng, ring: Ring): boolean {
  if (lat < ring.south || lat > ring.north || lng < ring.west || lng > ring.east) return false;
  let inside = false;
  const pts = ring.points;
  for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) {
    const [xi, yi] = pts[i];
    const [xj, yj] = pts[j];
    if (yi > lat !== yj > lat && lng < ((xj - xi) * (lat - yi)) / (yj - yi) + xi) {
      inside = !inside;
    }
  }
  return inside;
}

function insideAny(point: Latlng, rings: readonly Ring[]): boolean {
  return rings.some((ring) => insideRing(point, ring));
}

/** Points along a path, close enough together that no county-sized area slips between two. */
export function samplePath(path: readonly Latlng[]): Latlng[] {
  if (path.length <= 1) return [...path];
  const samples: Latlng[] = [path[0]];
  for (let i = 0; i < path.length - 1; i++) {
    const a = path[i];
    const b = path[i + 1];
    const span = Math.max(Math.abs(b.lat - a.lat), Math.abs(b.lng - a.lng));
    const steps = Math.min(MAX_SAMPLES_PER_LEG, Math.max(1, Math.ceil(span / SAMPLE_STEP_DEGREES)));
    for (let s = 1; s <= steps; s++) {
      const t = s / steps;
      samples.push({ lat: a.lat + (b.lat - a.lat) * t, lng: a.lng + (b.lng - a.lng) * t });
    }
  }
  return samples;
}

/**
 * Which shipments each alert touches, and which alerts each shipment runs
 * into. A shipment's hits are most severe first, and within a severity the
 * alert it is already inside comes before one still ahead.
 */
export function matchShipmentsToAlerts(
  shipments: readonly MapShipment[],
  features: readonly WeatherAlertFeature[],
): AlertMatches {
  const byShipment = new Map<string, AlertHit[]>();
  const byAlert = new Map<string, AlertHit[]>();
  const alerts = features
    .map((feature) => {
      const rings = alertOuterRings(feature).map(toRing);
      return {
        id: feature.properties.id,
        rank: severityRank(feature.properties.severity),
        rings,
        south: Math.min(...rings.map((r) => r.south)),
        north: Math.max(...rings.map((r) => r.north)),
        west: Math.min(...rings.map((r) => r.west)),
        east: Math.max(...rings.map((r) => r.east)),
      };
    })
    .filter((alert) => alert.id && alert.rings.length > 0);
  if (alerts.length === 0) return { byShipment, byAlert };

  for (const model of shipments) {
    const ahead = remainingPath(model);
    if (ahead.length === 0) continue;
    // A route's box rules out most alerts before a single point is sampled.
    let south = 90;
    let north = -90;
    let west = 180;
    let east = -180;
    for (const { lat, lng } of [model.position, ...ahead]) {
      south = Math.min(south, lat);
      north = Math.max(north, lat);
      west = Math.min(west, lng);
      east = Math.max(east, lng);
    }
    const nearby = alerts.filter(
      (alert) =>
        alert.south <= north && alert.north >= south && alert.west <= east && alert.east >= west,
    );
    if (nearby.length === 0) continue;
    const samples = samplePath(ahead);
    const hits: (AlertHit & { rank: number })[] = [];

    for (const alert of nearby) {
      let at: AlertHit["at"] | null = null;
      if (insideAny(model.position, alert.rings)) at = "stop";
      else if (samples.some((point) => insideAny(point, alert.rings))) at = "route";
      if (!at) continue;
      hits.push({ alertId: alert.id, shipmentId: model.id, at, rank: alert.rank });
    }
    if (hits.length === 0) continue;

    hits.sort((a, b) => a.rank - b.rank || (a.at === b.at ? 0 : a.at === "stop" ? -1 : 1));
    const clean = hits.map(({ alertId, shipmentId, at }) => ({ alertId, shipmentId, at }));
    byShipment.set(model.id, clean);
    for (const hit of clean) {
      const list = byAlert.get(hit.alertId);
      if (list) list.push(hit);
      else byAlert.set(hit.alertId, [hit]);
    }
  }

  return { byShipment, byAlert };
}

/** The box around an alert's area, or null when it has none. */
export function alertBounds(feature: WeatherAlertFeature): google.maps.LatLngBoundsLiteral | null {
  const rings = alertOuterRings(feature).map(toRing);
  if (rings.length === 0) return null;
  return {
    south: Math.min(...rings.map((r) => r.south)),
    north: Math.max(...rings.map((r) => r.north)),
    west: Math.min(...rings.map((r) => r.west)),
    east: Math.max(...rings.map((r) => r.east)),
  };
}
