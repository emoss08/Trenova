import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect, useRef } from "react";
import { GEOFENCE_OPACITY, GEOFENCE_STYLE, GEOFENCE_Z_INDEX } from "./geofence-styles";
import type { NormalizedGeofence } from "./geofence-types";

type Props = {
  geofence: Extract<NormalizedGeofence, { kind: "polygon" }>;
  onSelect: (geofence: NormalizedGeofence) => void;
};

export function GeofencePolygon({ geofence, onSelect }: Props) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  useEffect(() => {
    if (!map || !mapsLib) return;

    const polygon = new mapsLib.Polygon({
      paths: geofence.path,
      map,
      strokeColor: GEOFENCE_STYLE.stroke,
      strokeWeight: GEOFENCE_OPACITY.weight,
      strokeOpacity: GEOFENCE_OPACITY.stroke,
      fillColor: GEOFENCE_STYLE.fill,
      fillOpacity: GEOFENCE_OPACITY.fill,
      clickable: true,
      zIndex: GEOFENCE_Z_INDEX,
    });

    const overListener = polygon.addListener("mouseover", () => {
      polygon.setOptions({
        strokeOpacity: GEOFENCE_OPACITY.strokeHover,
        fillOpacity: GEOFENCE_OPACITY.fillHover,
        strokeWeight: GEOFENCE_OPACITY.weightHover,
      });
    });
    const outListener = polygon.addListener("mouseout", () => {
      polygon.setOptions({
        strokeOpacity: GEOFENCE_OPACITY.stroke,
        fillOpacity: GEOFENCE_OPACITY.fill,
        strokeWeight: GEOFENCE_OPACITY.weight,
      });
    });
    const clickListener = polygon.addListener("click", () => {
      onSelectRef.current(geofence);
    });

    return () => {
      overListener.remove();
      outListener.remove();
      clickListener.remove();
      polygon.setMap(null);
    };
  }, [map, mapsLib, geofence]);

  return null;
}
