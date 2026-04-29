import { LoadingSkeletonState } from "@/components/loading-skeleton";
import { Button } from "@/components/ui/button";
import { useMapId } from "@/hooks/use-map-id";
import { DEFAULT_ZOOM, GOOGLE_MAPS_ERROR_MESSAGE, US_CENTER } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { QueryErrorResetBoundary, useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { MapPinOffIcon, SettingsIcon, TriangleAlertIcon } from "lucide-react";
import { Suspense, useEffect, useMemo, useState } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { useNavigate } from "react-router";
import { GeofenceOverlay } from "./geofence-overlay";
import { GeofencePopover } from "./geofence-popover";
import { LocationAddressMarker } from "./location-address-marker";
import { MapControls } from "./map-controls";
import { MapZoomControls } from "./map-zoom-controls";
import { collectGeofencesFromLocations } from "./normalize-geofence";
import { OWMTileLayer, type OWMLayerId } from "./owm-tile-layer";
import { TrafficLayer } from "./traffic-layer";
import { useMapUIState } from "./use-map-ui-state";
import { WeatherAlertLayer } from "./weather-alert-layer";
import { WeatherRadarLayer } from "./weather-radar-layer";

const OWM_LAYER_MAP: Record<string, OWMLayerId> = {
  wind: "wind_new",
  clouds: "clouds_new",
  temperature: "temp_new",
  pressure: "pressure_new",
};

export default function ShipmentMapPanel() {
  const mapId = useMapId();
  const { data } = useSuspenseQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
  });
  const [selectedGeofenceId, setSelectedGeofenceId] = useState<string | null>(null);

  const locationsQuery = useQuery({
    ...queries.location.geofences(),
    staleTime: 5 * 60 * 1000,
  });
  const locations = useMemo(
    () => locationsQuery.data?.results ?? [],
    [locationsQuery.data],
  );
  const allGeofences = useMemo(
    () => collectGeofencesFromLocations(locations),
    [locations],
  );
  const locationsById = useMemo(() => {
    const lookup = new globalThis.Map<string, (typeof locations)[number]>();
    for (const loc of locations) {
      if (loc.id) lookup.set(loc.id, loc);
    }
    return lookup;
  }, [locations]);
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
  });

  const owmApiKey = owmQuery.data?.apiKey ?? "";

  const selectedGeofence = useMemo(
    () => allGeofences.find((g) => g.id === selectedGeofenceId) ?? null,
    [allGeofences, selectedGeofenceId],
  );

  const boundsPoints = useMemo<google.maps.LatLngLiteral[]>(() => {
    const points: google.maps.LatLngLiteral[] = [];
    for (const loc of locations) {
      if (
        loc.latitude != null &&
        loc.longitude != null &&
        Number.isFinite(loc.latitude) &&
        Number.isFinite(loc.longitude)
      ) {
        points.push({ lat: loc.latitude, lng: loc.longitude });
      }
    }
    for (const g of allGeofences) {
      if (g.kind === "polygon") {
        for (const p of g.path) points.push(p);
      }
    }
    return points;
  }, [locations, allGeofences]);

  useEffect(() => {
    if (!overlays.geofences) {
      setSelectedGeofenceId(null);
    }
  }, [overlays.geofences]);

  const owmLayerId = OWM_LAYER_MAP[weatherLayer];
  const showWeather = overlays.weather;

  return (
    <APIProvider apiKey={data.apiKey}>
      <div
        className={cn(
          "relative w-full overflow-hidden rounded-lg border",
          isFullscreen ? "fixed inset-0 z-50 h-screen rounded-none border-none" : "h-[400px]",
        )}
      >
        <Map
          mapId={mapId}
          mapTypeId={mapStyle}
          defaultCenter={US_CENTER}
          defaultZoom={DEFAULT_ZOOM}
          gestureHandling="greedy"
          disableDefaultUI
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
            <WeatherRadarLayer weatherLayer={weatherLayer} onWeatherLayerChange={setWeatherLayer} />
          )}
          {overlays.alerts && <WeatherAlertLayer />}
          <MapZoomControls />
          <MapControls
            mapStyle={mapStyle}
            onMapStyleChange={setMapStyle}
            overlays={overlays}
            onToggleOverlay={toggleOverlay}
            isFullscreen={isFullscreen}
            onToggleFullscreen={toggleFullscreen}
            boundsPoints={boundsPoints}
          />
        </Map>
      </div>
    </APIProvider>
  );
}

export function ShipmentMapPanelBoundary({ children }: { children: React.ReactNode }) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          fallbackRender={({ error }) => <MapErrorFallback error={error as Error} />}
          onReset={reset}
        >
          <Suspense fallback={<LoadingSkeletonState description="Loading map component..." />}>
            {children}
          </Suspense>
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}

function MapErrorFallback({ error }: { error: Error }) {
  const isConfigError = error.message === GOOGLE_MAPS_ERROR_MESSAGE;
  const navigate = useNavigate();

  return (
    <div className="relative h-[400px] w-full overflow-hidden rounded-lg border border-border">
      <img
        src="/integrations/empty-state/map-preview.webp"
        alt="Empty state map preview"
        className="absolute inset-0 size-full object-cover"
      />
      <div className="absolute inset-0 bg-background/70 backdrop-blur-sm" />
      <div className="relative flex size-full items-center justify-center">
        <div className="flex max-w-sm flex-col items-center gap-3 text-center">
          <div className="flex size-10 items-center justify-center rounded-lg border border-border bg-background">
            {isConfigError ? (
              <MapPinOffIcon className="size-5 text-muted-foreground" />
            ) : (
              <TriangleAlertIcon className="size-5 text-muted-foreground" />
            )}
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-foreground">
              {isConfigError ? "Map integration not configured" : "Unable to load map"}
            </p>
            <p className="text-xs text-muted-foreground">
              {isConfigError
                ? "A Google Maps API key is required to display the fleet map. Configure the integration to enable this feature."
                : "An error occurred while loading the map component. Please try refreshing the page."}
            </p>
          </div>
          {isConfigError && (
            <Button variant="outline" size="sm" onClick={() => navigate("/admin/integrations")}>
              <SettingsIcon className="size-3.5" />
              Configure Integration
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
