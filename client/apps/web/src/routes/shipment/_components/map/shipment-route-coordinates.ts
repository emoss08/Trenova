import { getOrderedStops } from "@/lib/shipment-utils";
import type { Shipment, Stop } from "@/types/shipment";

export type Latlng = google.maps.LatLngLiteral;
export type StopWithCoord = { stop: Stop; latlng: Latlng };

export function readLatLng(loc: Stop["location"] | null | undefined): Latlng | null {
  if (!loc || loc.latitude == null || loc.longitude == null) return null;
  if (!Number.isFinite(loc.latitude) || !Number.isFinite(loc.longitude)) return null;
  return { lat: loc.latitude, lng: loc.longitude };
}

export function getShipmentStopsWithCoords(shipment: Shipment): StopWithCoord[] {
  return getOrderedStops(shipment)
    .map((stop) => ({ stop, latlng: readLatLng(stop.location ?? null) }))
    .filter((x): x is StopWithCoord => x.latlng !== null);
}

export function pickCurrentStopIndex(stops: StopWithCoord[]): number {
  const inTransit = stops.findIndex((s) => s.stop.status === "InTransit");
  if (inTransit !== -1) return inTransit;

  let lastCompleted = -1;
  for (let i = 0; i < stops.length; i++) {
    if (stops[i].stop.status === "Completed") lastCompleted = i;
  }
  if (lastCompleted === -1) return 0;
  if (lastCompleted === stops.length - 1) return stops.length - 1;
  return lastCompleted + 1;
}

export function getShipmentCurrentLatLng(shipment: Shipment): Latlng | null {
  const stops = getShipmentStopsWithCoords(shipment);
  if (stops.length === 0) return null;

  return stops[pickCurrentStopIndex(stops)].latlng;
}
