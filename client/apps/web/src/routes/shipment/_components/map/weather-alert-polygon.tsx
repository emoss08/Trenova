import { ALERT_CATEGORY_CONFIG, type WeatherAlertFeature } from "@/types/weather-alert";
import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useEffect, useRef } from "react";
import { alertOuterRings } from "./weather-alert-matching";

type PolygonState = "idle" | "hovered" | "selected" | "dimmed";

const STATE_STYLE: Record<
  PolygonState,
  { strokeOpacity: number; fillOpacity: number; strokeWeight: number; zIndex: number }
> = {
  idle: { strokeOpacity: 0.8, fillOpacity: 0.13, strokeWeight: 1.25, zIndex: 10 },
  hovered: { strokeOpacity: 0.95, fillOpacity: 0.24, strokeWeight: 2, zIndex: 11 },
  selected: { strokeOpacity: 1, fillOpacity: 0.3, strokeWeight: 2.5, zIndex: 12 },
  dimmed: { strokeOpacity: 0.3, fillOpacity: 0.05, strokeWeight: 1, zIndex: 9 },
};

/**
 * One alert's area. The polygons are drawn once per alert and restyled in
 * place as the pointer and the selection move.
 */
export function WeatherAlertPolygon({
  feature,
  state,
  onClick,
  onHoverChange,
}: {
  feature: WeatherAlertFeature;
  state: PolygonState;
  onClick: () => void;
  onHoverChange: (hovered: boolean) => void;
}) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");
  const polygonsRef = useRef<google.maps.Polygon[]>([]);
  const click = useLatestCallback(onClick);
  const hover = useLatestCallback(onHoverChange);
  // Alerts refetch every two minutes as new objects; the areas are redrawn only
  // when an alert itself changed, not each time the list arrives again.
  const readFeature = useLatestCallback(() => feature);
  const revision = `${feature.properties.id}:${feature.properties.lastUpdatedAt}`;

  useEffect(() => {
    if (!map || !mapsLib) return;
    const feature = readFeature();
    const config =
      ALERT_CATEGORY_CONFIG[feature.properties.alertCategory] ?? ALERT_CATEGORY_CONFIG.other;

    const polygons = alertOuterRings(feature).map((ring) => {
      const polygon = new mapsLib.Polygon({
        paths: ring.map(([lng, lat]) => ({ lat, lng })),
        map,
        strokeColor: config.stroke,
        fillColor: config.stroke,
        ...STATE_STYLE.idle,
      });
      polygon.addListener("mouseover", () => hover(true));
      polygon.addListener("mouseout", () => hover(false));
      polygon.addListener("click", () => click());
      return polygon;
    });
    polygonsRef.current = polygons;

    return () => {
      for (const polygon of polygons) {
        google.maps.event.clearInstanceListeners(polygon);
        polygon.setMap(null);
      }
      polygonsRef.current = [];
    };
  }, [map, mapsLib, revision, readFeature, click, hover]);

  useEffect(() => {
    for (const polygon of polygonsRef.current) polygon.setOptions(STATE_STYLE[state]);
  }, [map, mapsLib, revision, state]);

  return null;
}
