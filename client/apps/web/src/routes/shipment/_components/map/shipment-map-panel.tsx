import { useT } from "@trenova/shared/i18n/use-t";
import { useMapStyle } from "@/hooks/use-map-style";
import { DEFAULT_ZOOM, US_CENTER } from "@trenova/shared/lib/constants";
import { integration } from "@/lib/queries/integration";
import { location as locationQueries } from "@/lib/queries/location";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { useEffect, useRef, useState, type RefObject } from "react";
import { useShipmentBoardStore } from "../board/store";
import { GeofenceOverlay } from "./geofence-overlay";
import { GeofencePopover } from "./geofence-popover";
import { LocationAddressMarker } from "./location-address-marker";
import { MapControls } from "./map-controls";
import { useMapFocusStore } from "./map-focus-store";
import { buildMapShipments, countByStage, type MapShipment } from "./map-shipment-model";
import { MapZoomControls } from "@/components/map/map-zoom-controls";
import { collectGeofencesFromLocations } from "./normalize-geofence";
import { OWMTileLayer, type OWMLayerId } from "./owm-tile-layer";
import { HoveredRoute, ShipmentFocusRoutes, type RouteRequest } from "./shipment-focus-routes";
import { ShipmentLaneOverlay } from "./shipment-lane-overlay";
import { ShipmentMapCard } from "./shipment-map-card";
import { ShipmentMapHeader } from "./shipment-map-header";
import { ShipmentMapRail } from "./shipment-map-rail";
import { ShipmentMarksOverlay } from "./shipment-marks-overlay";
import { TrafficLayer } from "./traffic-layer";
import { FocusAutoPan } from "./use-focus-pan";
import { useMapShipments } from "./use-map-shipments";
import { useMapUIState } from "./use-map-ui-state";
import { useWeatherAlerts } from "./use-weather-alerts";
import { VehiclePositionOverlay } from "./vehicle-position-overlay";
import { WeatherAlertLayer } from "./weather-alert-layer";
import { matchShipmentsToAlerts } from "./weather-alert-matching";
import { WeatherRadarLayer } from "./weather-radar-layer";
import "./shipment-map.css";

const LIVE_MAP_INSTANCE_ID = "shipment-live-map";

/** The selected alert draws the routes of this many of the loads it touches. */
const MAX_ALERT_CONTEXT_ROUTES = 8;

const OWM_LAYER_MAP: Record<string, OWMLayerId> = {
  wind: "wind_new",
  clouds: "clouds_new",
  temperature: "temp_new",
  pressure: "pressure_new",
};

/**
 * The live shipment map: one mark per active shipment with a route only for
 * what is in focus, a rail listing the shipments (and, with public alerts on,
 * the alerts that touch them), and one detail card.
 *
 * `fill` takes the height of the container, as the dispatch board does; without
 * it the panel keeps its own height, for the Home widget. The rail shows only
 * when the panel is wide enough to keep the map usable beside it.
 */
export default function ShipmentMapPanel({
  backgroundEnabled = true,
  fill = false,
}: {
  backgroundEnabled?: boolean;
  fill?: boolean;
}) {
  const t = useT();

  const { mapId, colorScheme } = useMapStyle();
  const { data } = useSuspenseQuery({
    ...integration.runtimeConfig("GoogleMaps"),
  });
  const mapShipmentsQuery = useMapShipments(backgroundEnabled);
  const [selectedGeofenceId, setSelectedGeofenceId] = useState<string | null>(null);

  const locationsQuery = useQuery({
    ...locationQueries.geofences(),
    staleTime: 5 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: backgroundEnabled,
  });
  const locations = locationsQuery.data?.results ?? [];
  const allGeofences = collectGeofencesFromLocations(locations);
  const locationsById = new globalThis.Map<string, (typeof locations)[number]>();
  for (const loc of locations) {
    if (loc.id) {
      locationsById.set(loc.id, loc);
    }
  }
  const {
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
  } = useMapUIState();

  const owmQuery = useQuery({
    ...integration.runtimeConfig("OpenWeatherMap"),
    staleTime: 5 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: backgroundEnabled,
  });

  const owmApiKey = owmQuery.data?.config.apiKey ?? "";

  const selectedGeofence = allGeofences.find((g) => g.id === selectedGeofenceId) ?? null;

  const allModels = buildMapShipments(mapShipmentsQuery.data?.results ?? []);
  const models = allModels.filter((m) => !hiddenStages.includes(m.stage));
  const modelsById = new globalThis.Map<string, MapShipment>(models.map((m) => [m.id, m]));

  const allAlerts = useWeatherAlerts(backgroundEnabled && overlays.alerts);
  const alerts = allAlerts.filter(
    (f) => !hiddenAlertCategories.includes(f.properties.alertCategory),
  );
  const alertsById = new globalThis.Map(alerts.map((f) => [f.properties.id, f]));
  const matches = matchShipmentsToAlerts(models, alerts);

  const focus = useMapFocusStore((s) => s.focus);
  const clearFocus = useMapFocusStore((s) => s.clearFocus);
  const resetFocus = useMapFocusStore((s) => s.reset);
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => resetFocus, [resetFocus]);

  useEffect(() => {
    if (!focus) return;
    const handler = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !event.defaultPrevented) clearFocus();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [focus, clearFocus]);

  // What stays in focus until the selection changes: the selected shipment, or
  // the loads the selected alert touches. Hover is kept out of this on purpose;
  // the marks, the dimming and the preview route each read it themselves, so a
  // pointer crossing the map redraws only those.
  const alertHits = focus?.kind === "alert" ? (matches.byAlert.get(focus.id) ?? []) : [];
  const pinnedIds = new Set<string>(alertHits.map((hit) => hit.shipmentId));
  const selected = focus?.kind === "shipment" ? modelsById.get(focus.id) : undefined;
  if (selected) pinnedIds.add(selected.id);

  const routes: RouteRequest[] = [];
  if (routeMode !== "off") {
    for (const hit of alertHits.slice(0, MAX_ALERT_CONTEXT_ROUTES)) {
      const model = modelsById.get(hit.shipmentId);
      if (model) routes.push({ model, emphasis: "context" });
    }
    if (selected) routes.push({ model: selected, emphasis: "selected" });
  }

  const boundsPoints: google.maps.LatLngLiteral[] = models.map((m) => m.position);
  for (const loc of locations) {
    if (
      loc.latitude != null &&
      loc.longitude != null &&
      Number.isFinite(loc.latitude) &&
      Number.isFinite(loc.longitude)
    ) {
      boundsPoints.push({ lat: loc.latitude, lng: loc.longitude });
    }
  }
  for (const g of allGeofences) {
    if (g.kind === "polygon") {
      boundsPoints.push(...g.path);
    }
  }

  const handleToggleOverlay = (overlay: Parameters<typeof toggleOverlay>[0]) => {
    if (overlay === "geofences" && overlays.geofences) {
      setSelectedGeofenceId(null);
    }
    if (overlay === "alerts" && overlays.alerts && focus?.kind === "alert") {
      clearFocus();
    }
    toggleOverlay(overlay);
  };

  const owmLayerId = OWM_LAYER_MAP[weatherLayer];
  const showWeather = overlays.weather;

  if (!data.config.apiKey) {
    return (
      <div
        className={cn(
          "bg-card text-muted-foreground flex items-center justify-center rounded-lg border px-6 text-center text-sm",
          isFullscreen ? "fixed inset-0 z-50 h-screen rounded-none border-none" : "h-full",
        )}
      >
        {t(
          "Google Maps is not configured for this environment, so the live shipment map cannot be displayed.",
        )}
      </div>
    );
  }

  return (
    <APIProvider apiKey={data.config.apiKey}>
      <div
        className={cn(
          "@container/map bg-card relative flex w-full flex-col overflow-hidden rounded-lg border",
          isFullscreen
            ? "fixed inset-0 z-50 h-screen rounded-none border-none"
            : fill
              ? "h-full min-h-105"
              : "h-[clamp(420px,calc(100vh-380px),540px)]",
        )}
      >
        <ShipmentMapHeader
          counts={countByStage(allModels)}
          hiddenStages={hiddenStages}
          onToggleStage={toggleStage}
          routeMode={routeMode}
          onRouteModeChange={setRouteMode}
          dataUpdatedAt={mapShipmentsQuery.dataUpdatedAt}
          controls={
            <MapControls
              mapStyle={mapStyle}
              onMapStyleChange={setMapStyle}
              overlays={overlays}
              onToggleOverlay={handleToggleOverlay}
              isFullscreen={isFullscreen}
              onToggleFullscreen={toggleFullscreen}
              boundsPoints={boundsPoints}
              mapInstanceId={LIVE_MAP_INSTANCE_ID}
            />
          }
        />
        <div className="flex min-h-0 flex-1">
          <ShipmentMapRail
            className="hidden @3xl/map:flex"
            models={models}
            matches={matches}
            alertsOn={overlays.alerts}
            features={alerts}
            allFeatures={allAlerts}
            hiddenAlertCategories={hiddenAlertCategories}
            onToggleAlertCategory={toggleAlertCategory}
          />
          <div ref={bodyRef} className="relative min-h-0 min-w-0 flex-1">
            <MapDimState target={bodyRef} pinned={pinnedIds.size > 0} modelsById={modelsById} />
            <Map
              id={LIVE_MAP_INSTANCE_ID}
              mapId={mapId}
              colorScheme={colorScheme}
              mapTypeId={mapStyle}
              defaultCenter={US_CENTER}
              defaultZoom={DEFAULT_ZOOM}
              gestureHandling="greedy"
              disableDefaultUI
              onClick={() => {
                if (useMapFocusStore.getState().focus) clearFocus();
              }}
              style={{ width: "100%", height: "100%" }}
            >
              {overlays.geofences &&
                allGeofences.map((geofence) => (
                  <GeofenceOverlay
                    key={geofence.id}
                    geofence={geofence}
                    onSelect={(g) => setSelectedGeofenceId(g.id)}
                  />
                ))}
              {overlays.geofences && selectedGeofence && (
                <GeofencePopover
                  geofence={selectedGeofence}
                  location={locationsById.get(selectedGeofence.id) ?? null}
                  onClose={() => setSelectedGeofenceId(null)}
                />
              )}
              {overlays.addresses &&
                locations.map((location) => (
                  <LocationAddressMarker key={location.id} location={location} />
                ))}
              {overlays.traffic && <TrafficLayer />}
              {showWeather && owmLayerId && owmApiKey && (
                <OWMTileLayer layerId={owmLayerId} apiKey={owmApiKey} />
              )}
              {showWeather && (
                <WeatherRadarLayer
                  weatherLayer={weatherLayer}
                  onWeatherLayerChange={setWeatherLayer}
                />
              )}
              {overlays.alerts && <WeatherAlertLayer features={alerts} />}
              {routeMode === "lanes" && (
                <ShipmentLaneOverlay models={models} dimmed={pinnedIds.size > 0} />
              )}
              <ShipmentFocusRoutes routes={routes} />
              {routeMode !== "off" ? (
                <HoveredRoute modelsById={modelsById} selectedId={selected?.id ?? null} />
              ) : null}
              <ShipmentMarksOverlay models={models} pinnedIds={pinnedIds} />
              <VehiclePositionOverlay enabled={backgroundEnabled && overlays.vehicles} />
              <FocusAutoPan
                mapInstanceId={LIVE_MAP_INSTANCE_ID}
                models={models}
                features={alerts}
                matches={matches}
              />
              <MapZoomControls />
            </Map>
            <ShipmentMapCard modelsById={modelsById} featuresById={alertsById} matches={matches} />
          </div>
        </div>
      </div>
    </APIProvider>
  );
}

/**
 * Sets `data-map-dim` on the map while a shipment is pinned or one on the map
 * is hovered; marks not in focus step back through `shipment-map.css`. An
 * attribute written here costs one element per hover, where a prop would
 * redraw every mark.
 */
function MapDimState({
  target,
  pinned,
  modelsById,
}: {
  target: RefObject<HTMLDivElement | null>;
  pinned: boolean;
  modelsById: ReadonlyMap<string, MapShipment>;
}) {
  const hovering = useShipmentBoardStore((s) => !!s.highlightId && modelsById.has(s.highlightId));
  const dim = pinned || hovering;

  useEffect(() => {
    target.current?.setAttribute("data-map-dim", String(dim));
  }, [target, dim]);

  return null;
}
