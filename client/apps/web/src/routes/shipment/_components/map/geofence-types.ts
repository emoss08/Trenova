import type { LocationGeofenceType } from "@trenova/shared/types/location";
import { defineLabels } from "@trenova/shared/i18n/labels";

type Base = {
  id: string;
  locationName: string;
  sourceType: LocationGeofenceType;
};

export type NormalizedGeofence =
  | (Base & {
      kind: "circle";
      center: google.maps.LatLngLiteral;
      radiusMeters: number;
    })
  | (Base & {
      kind: "polygon";
      path: google.maps.LatLngLiteral[];
    });

export const GEOFENCE_KIND_LABEL: Record<NormalizedGeofence["kind"], string> = defineLabels({
  circle: "Circle",
  polygon: "Polygon",
});
