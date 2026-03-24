import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect } from "react";

import type { MockStop } from "./mock-data";

const CIRCLE_STYLES = {
  pickup: { strokeColor: "#3b82f6", fillColor: "#3b82f680" },
  delivery: { strokeColor: "#16a34a", fillColor: "#16a34a80" },
} as const;

export function GeofenceCircle({ stop }: { stop: MockStop }) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");

  useEffect(() => {
    if (!map || !mapsLib) return;

    const style = CIRCLE_STYLES[stop.type];
    const circle = new mapsLib.Circle({
      center: { lat: stop.lat, lng: stop.lng },
      radius: 1500,
      map,
      strokeColor: style.strokeColor,
      strokeWeight: 2,
      strokeOpacity: 0.8,
      fillColor: style.fillColor,
      fillOpacity: 0.2,
    });

    return () => {
      circle.setMap(null);
    };
  }, [map, mapsLib, stop]);

  return null;
}
