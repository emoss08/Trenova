import { LoadingSkeletonState } from "@/components/loading-skeleton";
import { Button } from "@/components/ui/button";
import { DEFAULT_ZOOM, GOOGLE_MAPS_ERROR_MESSAGE, MAP_ID, US_CENTER } from "@/lib/constants";
import { queries } from "@/lib/queries";
import type { ShipmentStatus } from "@/types/shipment";
import { QueryErrorResetBoundary, useSuspenseQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { MapPinOffIcon, SettingsIcon, TriangleAlertIcon } from "lucide-react";
import { Suspense, useCallback, useMemo, useState } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { Link } from "react-router";
import { GeofenceCircle } from "./geofence-circle";
import { MapFilterBar, type MapFilters } from "./map-filter-bar";
import { MOCK_TRACTORS } from "./mock-data";
import { RoutePolyline } from "./route-polyline";
import { TractorInfoWindow } from "./tractor-info-window";
import { TractorMarker } from "./tractor-marker";

const INITIAL_FILTERS: MapFilters = {
  delayedOnly: false,
  statuses: new Set<ShipmentStatus>(),
};

export default function ShipmentMapPanel() {
  const { data } = useSuspenseQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
  });
  const [selectedTractorId, setSelectedTractorId] = useState<string | null>(null);
  const [filters, setFilters] = useState<MapFilters>(INITIAL_FILTERS);

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

  return (
    <APIProvider apiKey={data.apiKey}>
      <div className="relative h-[400px] w-full overflow-hidden rounded-lg border">
        <Map
          mapId={MAP_ID}
          defaultCenter={US_CENTER}
          defaultZoom={DEFAULT_ZOOM}
          gestureHandling="greedy"
          disableDefaultUI
          onClick={handleMapClick}
        >
          {filteredTractors.map((tractor) => (
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
              <RoutePolyline path={selectedTractor.routePath} />
              {selectedTractor.stops.map((stop) => (
                <GeofenceCircle key={stop.id} stop={stop} />
              ))}
            </>
          )}

          <MapFilterBar
            filters={filters}
            onFiltersChange={setFilters}
            totalCount={MOCK_TRACTORS.length}
            filteredCount={filteredTractors.length}
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
            <Button variant="outline" size="sm" render={<Link to="/admin/integrations" />}>
              <SettingsIcon className="size-3.5" />
              Configure Integration
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
