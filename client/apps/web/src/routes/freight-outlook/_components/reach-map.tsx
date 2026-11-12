import { mapColor } from "@/components/map/map-color";
import { MapPolyline } from "@/components/map/map-polyline";
import { useMapStyle } from "@/hooks/use-map-style";
import { useRootTheme } from "@/hooks/use-root-theme";
import type { FreightAreaBounds, FreightReach } from "@/lib/graphql/freight-outlook";
import { integration } from "@/lib/queries/integration";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { AdvancedMarker } from "@/components/map/advanced-marker";
import { AdvancedMarkerAnchorPoint, APIProvider, Map, useMap, useMapsLibrary } from "@vis.gl/react-google-maps";
import { Fragment, useEffect } from "react";
import { reachBounds } from "./outlook-model";
import { ReachMapSkeleton } from "./reach-map-skeleton";

/** Room left around the area and its drivers when the map frames them, in pixels. */
const FIT_PADDING = 28;

/** The tile the area counts its loads and drivers in, outlined at its true size. */
function AreaFootprint({ bounds }: { bounds: FreightAreaBounds }) {
  const map = useMap();
  const mapsLib = useMapsLibrary("maps");
  const theme = useRootTheme();
  const { south, west, north, east } = bounds;

  useEffect(() => {
    if (!map || !mapsLib) return;
    const color = mapColor("--brand");
    const rectangle = new mapsLib.Rectangle({
      map,
      bounds: { south, west, north, east },
      strokeColor: color,
      strokeOpacity: 0.7,
      strokeWeight: 1.5,
      fillColor: color,
      fillOpacity: 0.1,
      clickable: false,
      zIndex: 1,
    });
    return () => rectangle.setMap(null);
  }, [map, mapsLib, south, west, north, east, theme]);

  return null;
}

/** Frames the area and its drivers again when either changes under a map already showing. */
function FrameReach({ view }: { view: google.maps.LatLngBoundsLiteral }) {
  const map = useMap();
  const { south, west, north, east } = view;

  useEffect(() => {
    map?.fitBounds({ south, west, north, east }, FIT_PADDING);
  }, [map, south, west, north, east]);

  return null;
}

/** A name on the map: a solid chip, so it reads over roads, labels and water alike. */
function MarkLabel({
  side,
  tone,
  children,
}: {
  side: "left" | "right" | "above";
  tone: "area" | "good" | "muted";
  children: string;
}) {
  return (
    <span
      className={cn(
        "absolute rounded-xs bg-card/85 px-1 whitespace-nowrap",
        side === "left" && "top-1/2 right-full mr-1 -translate-y-1/2",
        side === "right" && "top-1/2 left-full ml-1 -translate-y-1/2",
        side === "above" && "bottom-full left-1/2 mb-1 -translate-x-1/2",
        tone === "area" && "text-xs font-semibold text-foreground",
        tone === "good" && "text-2xs font-medium text-foreground",
        tone === "muted" && "text-2xs font-medium text-muted-foreground",
      )}
    >
      {children}
    </span>
  );
}

function DriverMark({ name, good, side }: { name: string; good: boolean; side: "left" | "right" }) {
  return (
    <span className="relative block size-2.5">
      <span
        className={cn(
          "absolute inset-0 rounded-full border-2 border-card",
          good ? "bg-success-foreground" : "bg-foreground-subtle",
        )}
      />
      <MarkLabel side={side} tone={good ? "good" : "muted"}>
        {name}
      </MarkLabel>
    </span>
  );
}

function AreaMark({ name }: { name: string }) {
  return (
    <span className="relative block size-3.5">
      <span className="absolute inset-0 rounded-full border-2 border-card bg-warning ring-1 ring-warning-foreground" />
      <MarkLabel side="above" tone="area">
        {name}
      </MarkLabel>
    </span>
  );
}

/**
 * Who can get to an area in time, on a map the dispatcher can zoom and pan: the area's tile,
 * and each driver with a line in, solid when they arrive in time and pay for the miles.
 */
export default function ReachMap({
  label,
  latitude,
  longitude,
  bounds,
  reach,
}: {
  label: string;
  latitude: number;
  longitude: number;
  bounds: FreightAreaBounds;
  reach: readonly FreightReach[];
}) {
  const t = useT();
  const { mapId, colorScheme } = useMapStyle();
  const config = useQuery(integration.runtimeConfig("GoogleMaps"));

  if (config.isPending) {
    return <ReachMapSkeleton />;
  }
  const apiKey = config.data?.config.apiKey;
  if (!apiKey) {
    return (
      <p className="text-xs text-foreground-subtle">
        {config.isError
          ? t("The map didn't load, so the drivers are listed without it.")
          : t("Google Maps isn't connected, so the drivers are listed without a map.")}
      </p>
    );
  }

  const center = { lat: latitude, lng: longitude };
  const view = reachBounds(bounds, reach);

  return (
    <div
      role="region"
      aria-label={t("Drivers who can reach {0}", label)}
      className="relative aspect-[3/2] w-full overflow-hidden rounded-lg bg-muted/40"
    >
      <APIProvider apiKey={apiKey}>
        <Map
          mapId={mapId}
          colorScheme={colorScheme}
          defaultBounds={{ ...view, padding: FIT_PADDING }}
          gestureHandling="greedy"
          disableDefaultUI
          clickableIcons={false}
          style={{ width: "100%", height: "100%" }}
        >
          <FrameReach view={view} />
          <AreaFootprint bounds={bounds} />
          {reach.map((row) => {
            if (row.reach === "Here") return null;
            const good = row.reach === "OnTime";
            const path = [{ lat: row.latitude, lng: row.longitude }, center];
            return (
              <Fragment key={row.workerId}>
                {good ? (
                  <MapPolyline path={path} colorVar="--card" weight={5} opacity={0.8} zIndex={3} />
                ) : null}
                <MapPolyline
                  path={path}
                  colorVar={good ? "--success-foreground" : "--foreground-faint"}
                  weight={good ? 2.5 : 1.5}
                  opacity={1}
                  dashed={!good}
                  zIndex={good ? 4 : 2}
                />
              </Fragment>
            );
          })}
          {reach.map((row) => {
            const good = row.reach === "OnTime" || row.reach === "Here";
            return (
              <AdvancedMarker
                key={row.workerId}
                position={{ lat: row.latitude, lng: row.longitude }}
                anchorPoint={AdvancedMarkerAnchorPoint.CENTER}
                title={row.workerName}
                zIndex={good ? 3 : 2}
              >
                <DriverMark
                  name={row.workerName.split(" ")[0] ?? row.workerName}
                  good={good}
                  side={row.longitude < longitude ? "left" : "right"}
                />
              </AdvancedMarker>
            );
          })}
          <AdvancedMarker
            position={center}
            anchorPoint={AdvancedMarkerAnchorPoint.CENTER}
            title={label}
            zIndex={4}
          >
            <AreaMark name={label.split(",")[0] ?? label} />
          </AdvancedMarker>
        </Map>
      </APIProvider>
    </div>
  );
}
