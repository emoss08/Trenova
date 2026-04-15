import { LoadingSkeletonState } from "@/components/loading-skeleton";
import { Button } from "@/components/ui/button";
import { useMapId } from "@/hooks/use-map-id";
import { DEFAULT_ZOOM, GOOGLE_MAPS_ERROR_MESSAGE, US_CENTER } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { queries } from "@/lib/queries";
import type { ShipmentStatus } from "@/types/shipment";
import { QueryErrorResetBoundary, useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { MapPinOffIcon, SettingsIcon, TriangleAlertIcon } from "lucide-react";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { useNavigate } from "react-router";
import { GeofenceCircle } from "./geofence-circle";
import { MapControls } from "./map-controls";
import type { MapFilters } from "./map-filter-bar";
import { MOCK_TRACTORS } from "./mock-data";
import { OWMTileLayer, type OWMLayerId } from "./owm-tile-layer";
import { RoutePolyline } from "./route-polyline";
import { TractorInfoWindow } from "./tractor-info-window";
import { TractorMarker } from "./tractor-marker";
import { TrafficLayer } from "./traffic-layer";
import { useMapUIState } from "./use-map-ui-state";
import { MapZoomControls } from "./map-zoom-controls";
import { WeatherRadarLayer } from "./weather-radar-layer";

const INITIAL_FILTERS: MapFilters = {
  delayedOnly: false,
  statuses: new Set<ShipmentStatus>(),
};

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
  const [selectedTractorId, setSelectedTractorId] = useState<string | null>(null);
  const [filters, setFilters] = useState<MapFilters>(INITIAL_FILTERS);
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

  const filteredTractors = useMemo(() => {
    return MOCK_TRACTORS.filter((t) => {
      if (filters.delayedOnly && t.shipmentStatus !== "Delayed") return false;
      if (filters.statuses.size > 0 && !filters.statuses.has(t.shipmentStatus)) return false;
      return true;
    });
  }, [filters]);

  const selectedTractor = useMemo(
    () => MOCK_TRACTORS.find((t) => t.id === selectedTractorId) ?? null,
    [selectedTractorId],
  );

  const handleMarkerClick = useCallback((id: string) => {
    setSelectedTractorId((prev) => (prev === id ? null : id));
  }, []);

  const handleMapClick = useCallback(() => {
    setSelectedTractorId(null);
  }, []);

  useEffect(() => {
    if (!overlays.vehicles) {
      setSelectedTractorId(null);
    }
  }, [overlays.vehicles]);

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
          onClick={handleMapClick}
        >
          {overlays.vehicles &&
            filteredTractors.map((tractor) => (
              <TractorMarker
                key={tractor.id}
                tractor={tractor}
                isSelected={tractor.id === selectedTractorId}
                onClick={handleMarkerClick}
              />
            ))}

          {selectedTractor && (
            <>
              <TractorInfoWindow
                tractor={selectedTractor}
                onClose={() => setSelectedTractorId(null)}
              />
              {overlays.routes && <RoutePolyline path={selectedTractor.routePath} />}
              {selectedTractor.stops.map((stop) => (
                <span key={stop.id}>
                  {overlays.geofences && <GeofenceCircle stop={stop} />}
                </span>
              ))}
            </>
          )}

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

          <MapZoomControls />

          <MapControls
            mapStyle={mapStyle}
            onMapStyleChange={setMapStyle}
            overlays={overlays}
            onToggleOverlay={toggleOverlay}
            isFullscreen={isFullscreen}
            onToggleFullscreen={toggleFullscreen}
            filteredTractors={filteredTractors}
            filters={filters}
            onFiltersChange={setFilters}
            totalCount={MOCK_TRACTORS.length}
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
