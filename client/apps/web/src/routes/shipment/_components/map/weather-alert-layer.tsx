import { queries } from "@/lib/queries";
import type { WeatherAlertFeature } from "@/types/weather-alert";
import { useQuery } from "@tanstack/react-query";
import { useMap } from "@vis.gl/react-google-maps";
import { useCallback, useState } from "react";
import { WeatherAlertDetailPanel } from "./weather-alert-detail-panel";
import { WeatherAlertLegendPanel } from "./weather-alert-legend-panel";
import { WeatherAlertPolygon } from "./weather-alert-polygon";

function getBoundsForFeature(feature: WeatherAlertFeature): google.maps.LatLngBoundsLiteral | null {
  if (!feature.geometry) return null;

  let allCoords: number[][] = [];
  if (feature.geometry.type === "Polygon") {
    const coords = feature.geometry.coordinates as number[][][];
    allCoords = coords[0] ?? [];
  } else if (feature.geometry.type === "MultiPolygon") {
    const coords = feature.geometry.coordinates as number[][][][];
    for (const polygon of coords) {
      if (polygon[0]) allCoords = allCoords.concat(polygon[0]);
    }
  }

  if (allCoords.length === 0) return null;

  let south = 90;
  let north = -90;
  let west = 180;
  let east = -180;

  for (const [lng, lat] of allCoords) {
    if (lat < south) south = lat;
    if (lat > north) north = lat;
    if (lng < west) west = lng;
    if (lng > east) east = lng;
  }

  return { south, north, west, east };
}

export function WeatherAlertLayer() {
  const map = useMap();
  const { data } = useQuery({
    ...queries.weatherAlert.alerts(),
    staleTime: 2 * 60 * 1000,
    refetchInterval: 2 * 60 * 1000,
  });

  const [selectedFeature, setSelectedFeature] = useState<WeatherAlertFeature | null>(null);
  const [legendCollapsed, setLegendCollapsed] = useState(false);

  const handlePolygonClick = useCallback(
    (feature: WeatherAlertFeature) => {
      setSelectedFeature((prev) => {
        if (prev?.properties.id === feature.properties.id) return null;

        const bounds = getBoundsForFeature(feature);
        if (bounds && map) {
          map.fitBounds(bounds, { top: 60, right: 60, bottom: 60, left: 340 });
        }

        setLegendCollapsed(true);
        return feature;
      });
    },
    [map],
  );

  const features = (data?.features ?? []).filter(
    (f) => f.geometry != null && f.properties.id != null,
  );

  return (
    <>
      {features.map((feature) => (
        <WeatherAlertPolygon
          key={feature.properties.id}
          feature={feature}
          onClick={handlePolygonClick}
        />
      ))}

      <WeatherAlertLegendPanel
        features={features}
        collapsed={legendCollapsed}
        onCollapsedChange={setLegendCollapsed}
      />

      {selectedFeature && (
        <WeatherAlertDetailPanel
          alertId={selectedFeature.properties.id}
          feature={selectedFeature}
          onClose={() => setSelectedFeature(null)}
        />
      )}
    </>
  );
}
