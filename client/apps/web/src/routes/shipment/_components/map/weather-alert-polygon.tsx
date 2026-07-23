import { ALERT_CATEGORY_CONFIG, type WeatherAlertFeature } from "@/types/weather-alert";
import { useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { useEffect, useRef } from "react";

function coordsToLatLng(ring: number[][]): google.maps.LatLngLiteral[] {
  return ring.map(([lng, lat]) => ({ lat, lng }));
}

export function WeatherAlertPolygon({
  feature,
  onClick,
}: {
  feature: WeatherAlertFeature;
  onClick: (feature: WeatherAlertFeature) => void;
}) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");
  const onClickRef = useRef(onClick);
  onClickRef.current = onClick;

  useEffect(() => {
    if (!map || !mapsLib || !feature.geometry) return;

    const config =
      ALERT_CATEGORY_CONFIG[feature.properties.alertCategory] ?? ALERT_CATEGORY_CONFIG.other;
    const polygons: google.maps.Polygon[] = [];

    const createPolygon = (paths: google.maps.LatLngLiteral[]) => {
      if (paths.length === 0) return;

      const polygon = new mapsLib.Polygon({
        paths,
        map,
        strokeColor: config.stroke,
        strokeWeight: 1.5,
        strokeOpacity: 0.7,
        fillColor: config.stroke,
        fillOpacity: 0.15,
        zIndex: 10,
      });

      polygon.addListener("mouseover", () => {
        polygon.setOptions({ strokeOpacity: 0.9, fillOpacity: 0.2, strokeWeight: 2 });
      });

      polygon.addListener("mouseout", () => {
        polygon.setOptions({ strokeOpacity: 0.7, fillOpacity: 0.15, strokeWeight: 1.5 });
      });

      polygon.addListener("click", () => {
        onClickRef.current(feature);
      });

      polygons.push(polygon);
    };

    const geomType = feature.geometry.type;
    if (geomType === "Polygon") {
      const coords = feature.geometry.coordinates as number[][][];
      if (coords[0]) createPolygon(coordsToLatLng(coords[0]));
    } else if (geomType === "MultiPolygon") {
      const coords = feature.geometry.coordinates as number[][][][];
      for (const polygon of coords) {
        if (polygon[0]) createPolygon(coordsToLatLng(polygon[0]));
      }
    }

    return () => {
      for (const polygon of polygons) {
        polygon.setMap(null);
      }
    };
  }, [map, mapsLib, feature]);

  return null;
}
