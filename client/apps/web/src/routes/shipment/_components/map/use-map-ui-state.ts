import { useLocalStorage } from "@/hooks/use-local-storage";
import { useShipmentMapStore } from "@/stores/shipment-map-store";
import {
  ROUTE_MODES,
  type MapStyleId,
  type OverlayId,
  type RouteMode,
  type WeatherLayerId,
} from "@/types/shipment-map";
import { ALERT_CATEGORY_CONFIG, type WeatherAlertCategory } from "@/types/weather-alert";
import { useCallback, useEffect, useMemo } from "react";
import { DEFAULT_HIDDEN_STAGES, isMapStage, type MapStage } from "./map-shipment-model";

const DEFAULT_OVERLAYS: Record<OverlayId, boolean> = {
  vehicles: true,
  geofences: false,
  addresses: false,
  traffic: false,
  weather: false,
  alerts: false,
};

function isRouteMode(value: unknown): value is RouteMode {
  return typeof value === "string" && (ROUTE_MODES as readonly string[]).includes(value);
}

function isAlertCategory(value: unknown): value is WeatherAlertCategory {
  return typeof value === "string" && value in ALERT_CATEGORY_CONFIG;
}

/* Stored values come back from another build of the app as often as from this
   one, so anything the current build does not know is dropped. */
function knownValues<T>(stored: unknown, guard: (value: unknown) => value is T): T[] {
  return Array.isArray(stored) ? stored.filter(guard) : [];
}

function toggled<T>(list: readonly T[], value: T): T[] {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value];
}

export function useMapUIState() {
  const [rawOverlays, setOverlays] = useLocalStorage<Record<OverlayId, boolean>>(
    "shipment-map-overlays",
    DEFAULT_OVERLAYS,
  );

  const overlays = useMemo(() => ({ ...DEFAULT_OVERLAYS, ...rawOverlays }), [rawOverlays]);
  const [mapStyle, setMapStyle] = useLocalStorage<MapStyleId>("shipment-map-style", "roadmap");
  const [weatherLayer, setWeatherLayer] = useLocalStorage<WeatherLayerId>(
    "shipment-map-weather",
    "precipitation",
  );
  const [rawRouteMode, setRouteMode] = useLocalStorage<RouteMode>(
    "shipment-map-route-mode",
    "selected",
  );
  const [rawHiddenStages, setHiddenStages] = useLocalStorage<MapStage[]>(
    "shipment-map-hidden-stages",
    [...DEFAULT_HIDDEN_STAGES],
  );
  const [rawHiddenAlertCategories, setHiddenAlertCategories] = useLocalStorage<
    WeatherAlertCategory[]
  >("shipment-map-hidden-alert-categories", []);
  const isFullscreen = useShipmentMapStore((s) => s.isFullscreen);
  const setIsFullscreen = useShipmentMapStore((s) => s.setIsFullscreen);

  const routeMode = isRouteMode(rawRouteMode) ? rawRouteMode : "selected";
  const hiddenStages = knownValues(rawHiddenStages, isMapStage);
  const hiddenAlertCategories = knownValues(rawHiddenAlertCategories, isAlertCategory);

  const toggleOverlay = useCallback(
    (id: OverlayId) => {
      setOverlays((prev) => ({ ...prev, [id]: !prev[id] }));
    },
    [setOverlays],
  );

  const toggleStage = useCallback(
    (stage: MapStage) => setHiddenStages((prev) => toggled(knownValues(prev, isMapStage), stage)),
    [setHiddenStages],
  );

  const toggleAlertCategory = useCallback(
    (category: WeatherAlertCategory) =>
      setHiddenAlertCategories((prev) => toggled(knownValues(prev, isAlertCategory), category)),
    [setHiddenAlertCategories],
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
    routeMode,
    setRouteMode,
    hiddenStages,
    toggleStage,
    hiddenAlertCategories,
    toggleAlertCategory,
    isFullscreen,
    toggleFullscreen,
  };
}
