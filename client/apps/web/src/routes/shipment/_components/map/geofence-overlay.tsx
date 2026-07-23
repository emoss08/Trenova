import { GeofenceCircle } from "./geofence-circle";
import { GeofencePolygon } from "./geofence-polygon";
import type { NormalizedGeofence } from "./geofence-types";

type Props = {
  geofence: NormalizedGeofence;
  onSelect: (geofence: NormalizedGeofence) => void;
};

export function GeofenceOverlay({ geofence, onSelect }: Props) {
  return geofence.kind === "circle" ? (
    <GeofenceCircle geofence={geofence} onSelect={onSelect} />
  ) : (
    <GeofencePolygon geofence={geofence} onSelect={onSelect} />
  );
}
