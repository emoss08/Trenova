import { LoadingSkeletonState } from "@trenova/shared/components/loading-skeleton";
import { Button } from "@trenova/shared/components/ui/button";
import { GOOGLE_MAPS_ERROR_MESSAGE } from "@trenova/shared/lib/constants";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import { MapPinOffIcon, SettingsIcon, TriangleAlertIcon } from "lucide-react";
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { useNavigate } from "react-router";

export function ShipmentMapPanelBoundary({ children }: { children: React.ReactNode }) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          fallbackRender={({ error }) => <MapErrorFallback error={error as Error} />}
          onReset={reset}
        >
          <Suspense
            fallback={
              <LoadingSkeletonState description="Loading map component..." className="h-full" />
            }
          >
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
    <div className="relative h-[clamp(420px,calc(100vh-380px),540px)] w-full overflow-hidden rounded-lg border border-border">
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
