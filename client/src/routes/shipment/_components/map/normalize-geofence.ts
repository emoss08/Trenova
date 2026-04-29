import type {
  Location,
  LocationGeofenceType,
  LocationGeofenceVertex,
} from "@/types/location";
import type { NormalizedGeofence } from "./geofence-types";

export type GeofenceInput = {
  id: string;
  locationName: string;
  geofenceType: LocationGeofenceType;
  latitude?: number | null;
  longitude?: number | null;
  geofenceRadiusMeters?: number | null;
  geofenceVertices?: LocationGeofenceVertex[] | null;
};

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function toLatLng(vertex: LocationGeofenceVertex): google.maps.LatLngLiteral | null {
  if (!isFiniteNumber(vertex?.latitude) || !isFiniteNumber(vertex?.longitude)) {
    return null;
  }
  return { lat: vertex.latitude, lng: vertex.longitude };
}

type Base = Pick<NormalizedGeofence, "id" | "locationName" | "sourceType">;

function buildPolygon(base: Base, vertices: LocationGeofenceVertex[]): NormalizedGeofence | null {
  const path: google.maps.LatLngLiteral[] = [];
  for (const vertex of vertices) {
    const point = toLatLng(vertex);
    if (point) path.push(point);
  }
  if (path.length < 3) return null;
  return { ...base, kind: "polygon", path };
}

function buildCircle(
  base: Base,
  center: google.maps.LatLngLiteral,
  radiusMeters: number,
): NormalizedGeofence {
  return { ...base, kind: "circle", center, radiusMeters };
}

export function normalizeGeofence(input: GeofenceInput): NormalizedGeofence | null {
  const base: Base = {
    id: input.id,
    locationName: input.locationName,
    sourceType: input.geofenceType,
  };

  const vertices = input.geofenceVertices ?? [];
  const hasUsableCircle =
    isFiniteNumber(input.latitude) &&
    isFiniteNumber(input.longitude) &&
    isFiniteNumber(input.geofenceRadiusMeters) &&
    (input.geofenceRadiusMeters as number) > 0;

  switch (input.geofenceType) {
    case "circle":
      if (!hasUsableCircle) return null;
      return buildCircle(
        base,
        { lat: input.latitude as number, lng: input.longitude as number },
        input.geofenceRadiusMeters as number,
      );

    case "rectangle":
    case "draw":
      return buildPolygon(base, vertices);

    case "auto":
      if (vertices.length >= 3) {
        const polygon = buildPolygon(base, vertices);
        if (polygon) return polygon;
      }
      if (hasUsableCircle) {
        return buildCircle(
          base,
          { lat: input.latitude as number, lng: input.longitude as number },
          input.geofenceRadiusMeters as number,
        );
      }
      return null;

    default:
      return null;
  }
}

export function collectGeofencesFromLocations(locations: Location[]): NormalizedGeofence[] {
  const byLocation = new Map<string, NormalizedGeofence>();

  for (const loc of locations) {
    if (!loc?.id) continue;
    if (byLocation.has(loc.id)) continue;

    const normalized = normalizeGeofence({
      id: loc.id,
      locationName: loc.name,
      geofenceType: loc.geofenceType,
      latitude: loc.latitude ?? null,
      longitude: loc.longitude ?? null,
      geofenceRadiusMeters: loc.geofenceRadiusMeters ?? null,
      geofenceVertices: loc.geofenceVertices,
    });
    if (normalized) {
      byLocation.set(loc.id, normalized);
    }
  }

  return Array.from(byLocation.values());
}
