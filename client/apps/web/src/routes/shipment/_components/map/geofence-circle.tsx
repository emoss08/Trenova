import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect, useRef } from "react";
import { GEOFENCE_OPACITY, GEOFENCE_STYLE, GEOFENCE_Z_INDEX } from "./geofence-styles";
import type { NormalizedGeofence } from "./geofence-types";

type Props = {
  geofence: Extract<NormalizedGeofence, { kind: "circle" }>;
  onSelect: (geofence: NormalizedGeofence) => void;
};

export function GeofenceCircle({ geofence, onSelect }: Props) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  useEffect(() => {
    if (!map || !mapsLib) return;

    const circle = new mapsLib.Circle({
      center: geofence.center,
      radius: geofence.radiusMeters,
      map,
      strokeColor: GEOFENCE_STYLE.stroke,
      strokeWeight: GEOFENCE_OPACITY.weight,
      strokeOpacity: GEOFENCE_OPACITY.stroke,
      fillColor: GEOFENCE_STYLE.fill,
      fillOpacity: GEOFENCE_OPACITY.fill,
      clickable: true,
      zIndex: GEOFENCE_Z_INDEX,
    });

    const overListener = circle.addListener("mouseover", () => {
      circle.setOptions({
        strokeOpacity: GEOFENCE_OPACITY.strokeHover,
        fillOpacity: GEOFENCE_OPACITY.fillHover,
        strokeWeight: GEOFENCE_OPACITY.weightHover,
      });
    });
    const outListener = circle.addListener("mouseout", () => {
      circle.setOptions({
        strokeOpacity: GEOFENCE_OPACITY.stroke,
        fillOpacity: GEOFENCE_OPACITY.fill,
        strokeWeight: GEOFENCE_OPACITY.weight,
      });
    });
    const clickListener = circle.addListener("click", () => {
      onSelectRef.current(geofence);
    });

    return () => {
      overListener.remove();
      outListener.remove();
      clickListener.remove();
      circle.setMap(null);
    };
  }, [map, mapsLib, geofence]);

  return null;
}
