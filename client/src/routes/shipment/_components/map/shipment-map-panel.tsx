import { useMapId } from "@/hooks/use-map-id";
import { DEFAULT_ZOOM, US_CENTER } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { formatElapsedTime } from "@/lib/time-utils";
import { cn } from "@/lib/utils";
import { useRealtimeStore } from "@/stores/realtime-store";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { useEffect, useState } from "react";
import { GeofenceOverlay } from "./geofence-overlay";
import { GeofencePopover } from "./geofence-popover";
import { LocationAddressMarker } from "./location-address-marker";
import { MapControls } from "./map-controls";
import { MapZoomControls } from "./map-zoom-controls";
import { collectGeofencesFromLocations } from "./normalize-geofence";
import { OWMTileLayer, type OWMLayerId } from "./owm-tile-layer";
import { ShipmentMapLegend } from "./shipment-map-legend";
import { ShipmentRouteOverlay } from "./shipment-route-overlay";
import { TrafficLayer } from "./traffic-layer";
import { HighlightAutoPan } from "./use-highlight-pan";
import { useMapShipments } from "./use-map-shipments";
import { useMapUIState } from "./use-map-ui-state";
import { WeatherAlertLayer } from "./weather-alert-layer";
import { WeatherRadarLayer } from "./weather-radar-layer";

const LIVE_MAP_INSTANCE_ID = "shipment-live-map";

const OWM_LAYER_MAP: Record<string, OWMLayerId> = {
  wind: "wind_new",
  clouds: "clouds_new",
  temperature: "temp_new",
  pressure: "pressure_new",
};

export default function ShipmentMapPanel({
  backgroundEnabled = true,
}: {
  backgroundEnabled?: boolean;
}) {
  const mapId = useMapId();
  const { data } = useSuspenseQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
  });
  const mapShipmentsQuery = useMapShipments(backgroundEnabled);
  const [selectedGeofenceId, setSelectedGeofenceId] = useState<string | null>(null);

  const locationsQuery = useQuery({
    ...queries.location.geofences(),
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
    isFullscreen,
    toggleFullscreen,
  } = useMapUIState();

  const owmQuery = useQuery({
    ...queries.integration.runtimeConfig("OpenWeatherMap"),
    staleTime: 5 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: backgroundEnabled,
  });

  const owmApiKey = owmQuery.data?.config.apiKey ?? "";

  const selectedGeofence = allGeofences.find((g) => g.id === selectedGeofenceId) ?? null;

  const boundsPoints: google.maps.LatLngLiteral[] = [];
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
    toggleOverlay(overlay);
  };

  const owmLayerId = OWM_LAYER_MAP[weatherLayer];
  const showWeather = overlays.weather;
  const mapShipments = mapShipmentsQuery.data?.results ?? [];
  const delayedCount = mapShipments.filter((shipment) => shipment.status === "Delayed").length;
  const inTransitCount = mapShipments.filter(
    (shipment) => shipment.status === "InTransit" || shipment.status === "PartiallyCompleted",
  ).length;

  if (!data.config.apiKey) {
    return (
      <div
        className={cn(
          "flex items-center justify-center rounded-lg border bg-card px-6 text-center text-sm text-muted-foreground",
          isFullscreen ? "fixed inset-0 z-50 h-screen rounded-none border-none" : "h-full",
        )}
      >
        Google Maps is not configured for this environment, so the live shipment map cannot be
        displayed.
      </div>
    );
  }

  return (
    <APIProvider apiKey={data.config.apiKey}>
      <div
        className={cn(
          "relative flex w-full flex-col overflow-hidden rounded-lg border bg-card",
          isFullscreen
            ? "fixed inset-0 z-50 h-screen rounded-none border-none"
            : "h-[clamp(420px,calc(100vh-380px),540px)]",
        )}
      >
        <div className="flex h-9 shrink-0 items-center justify-between border-b border-border bg-card px-2.5">
          <div className="flex min-w-0 items-center gap-2">
            <span className="text-xs font-semibold text-foreground">Live Map</span>
            <span className="shrink truncate rounded-md border border-border bg-muted/60 px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
              {delayedCount} at-risk · {inTransitCount} in-transit
            </span>
          </div>
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
        </div>
        <div className="relative min-h-0 flex-1">
          <Map
            id={LIVE_MAP_INSTANCE_ID}
            mapId={mapId}
            mapTypeId={mapStyle}
            defaultCenter={US_CENTER}
            defaultZoom={DEFAULT_ZOOM}
            gestureHandling="greedy"
            disableDefaultUI
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
            {overlays.alerts && <WeatherAlertLayer />}
            <ShipmentRouteOverlay enabled={backgroundEnabled} />
            <HighlightAutoPan mapInstanceId={LIVE_MAP_INSTANCE_ID} enabled={backgroundEnabled} />
            <MapZoomControls />
          </Map>
          <LiveMapSyncOverlay
            unitCount={mapShipments.length}
            dataUpdatedAt={mapShipmentsQuery.dataUpdatedAt}
          />
          <ShipmentMapLegend />
        </div>
      </div>
    </APIProvider>
  );
}

function LiveMapSyncOverlay({
  unitCount,
  dataUpdatedAt,
}: {
  unitCount: number;
  dataUpdatedAt: number;
}) {
  const connectionState = useRealtimeStore.use.connectionState();
  const lastEventAt = useRealtimeStore.use.lastEventAt();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, []);

  const syncedAt = Math.max(lastEventAt ?? 0, dataUpdatedAt ?? 0);
  const live = connectionState === "connected";

  return (
    <div className="pointer-events-none absolute top-3 left-3 z-10 flex items-center gap-1.5">
      <span className="inline-flex items-center gap-1 rounded-md border border-border bg-card/80 px-2 py-1 font-mono text-[10px] font-medium text-foreground shadow-sm backdrop-blur-sm">
        <span
          aria-hidden
          className={cn(
            "size-1.5 rounded-full",
            live ? "animate-pulse bg-success" : "bg-muted-foreground",
          )}
        />
        {live ? "LIVE" : "OFFLINE"} · {unitCount} units
      </span>
      <span className="rounded-md border border-border bg-card/80 px-2 py-1 font-mono text-[10px] font-medium text-muted-foreground shadow-sm backdrop-blur-sm">
        synced {formatElapsedTime(syncedAt, now)}
      </span>
    </div>
  );
}
