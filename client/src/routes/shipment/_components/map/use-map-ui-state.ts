import { useLocalStorage } from "@/hooks/use-local-storage";
import { useShipmentMapStore } from "@/stores/shipment-map-store";
import type { MapStyleId, OverlayId, WeatherLayerId } from "@/types/shipment-map";
import { useCallback, useEffect, useMemo } from "react";

const DEFAULT_OVERLAYS: Record<OverlayId, boolean> = {
  vehicles: true,
  routes: true,
  stops: true,
  geofences: true,
  traffic: false,
  weather: false,
  alerts: false,
};

export function useMapUIState() {
  const [rawOverlays, setOverlays] = useLocalStorage<Record<OverlayId, boolean>>(
    "shipment-map-overlays",
    DEFAULT_OVERLAYS,
  );

  const overlays = useMemo(
    () => ({ ...DEFAULT_OVERLAYS, ...rawOverlays }),
    [rawOverlays],
  );
  const [mapStyle, setMapStyle] = useLocalStorage<MapStyleId>("shipment-map-style", "roadmap");
  const [weatherLayer, setWeatherLayer] = useLocalStorage<WeatherLayerId>(
    "shipment-map-weather",
    "precipitation",
  );
  const isFullscreen = useShipmentMapStore.use.isFullscreen();
  const setIsFullscreen = useShipmentMapStore.use.setIsFullscreen();

  const toggleOverlay = useCallback(
    (id: OverlayId) => {
      setOverlays((prev) => ({ ...prev, [id]: !prev[id] }));
    },
    [setOverlays],
  );

  const toggleFullscreen = useCallback(() => {
    setIsFullscreen((p) => !p);
  }, [setIsFullscreen]);

  useEffect(() => {
    if (!isFullscreen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") setIsFullscreen(false);
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [isFullscreen, setIsFullscreen]);

  return {
    overlays,
    toggleOverlay,
    mapStyle,
    setMapStyle,
    weatherLayer,
    setWeatherLayer,
    isFullscreen,
    toggleFullscreen,
  };
}
