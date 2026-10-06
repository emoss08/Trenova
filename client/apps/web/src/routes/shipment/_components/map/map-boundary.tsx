import { useT } from "@trenova/shared/i18n/use-t";
import { LoadingSkeletonState } from "@trenova/shared/components/loading-skeleton";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";

export function ShipmentMapPanelBoundary({ children }: { children: React.ReactNode }) {
  const t = useT();

  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary fallbackRender={() => <MapErrorFallback />} onReset={reset}>
          <Suspense
            fallback={
              <LoadingSkeletonState
                description={t("Loading map component...")}
                className="h-full"
              />
            }
          >
            {children}
          </Suspense>
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}

function MapErrorFallback() {
  const t = useT();

  return (
    <div className="border-border relative h-[clamp(420px,calc(100vh-380px),540px)] w-full overflow-hidden rounded-lg border">
      <img
        src="/integrations/empty-state/map-preview.webp"
        alt={t("Empty state map preview")}
        className="absolute inset-0 size-full object-cover"
      />
      <div className="bg-background/70 absolute inset-0 backdrop-blur-sm" />
      <div className="relative flex size-full items-center justify-center">
        <div className="flex max-w-sm flex-col items-center gap-3 text-center">
          <div className="border-border bg-background flex size-10 items-center justify-center rounded-lg border">
            <AlertTriangleIcon className="text-muted-foreground size-5" />
          </div>
          <div className="space-y-1">
            <p className="text-foreground text-sm font-medium">{t("Unable to load map")}</p>
            <p className="text-muted-foreground text-xs">
              {t(
                "An error occurred while loading the map component. Please try refreshing the page.",
              )}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
