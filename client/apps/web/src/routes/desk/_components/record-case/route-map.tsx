import { FitBounds } from "@/components/command-palette/_components/map-components";
import { MapPolyline } from "@/components/map/map-polyline";
import { useMapStyle } from "@/hooks/use-map-style";
import type { DeskCaseRoute } from "@/lib/graphql/desk-case";
import { AdvancedMarker, useMapDrawn, useMarkersUsable } from "@/components/map/advanced-marker";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { useT } from "@trenova/shared/i18n/use-t";
import { DEFAULT_ZOOM, US_CENTER } from "@trenova/shared/lib/constants";
import { cn } from "@trenova/shared/lib/utils";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useEffect, useMemo } from "react";

type LatLng = { lat: number; lng: number };

/**
 * The route on a map: the stops in order, the part already driven drawn
 * heavier, the truck where it last reported and a stop the chosen option
 * would add drawn dashed. It is read on demand: the map's library is large,
 * and a case is read many times without its evidence being looked at.
 */
export default function RouteMap({
  route,
  apiKey,
  onUnavailable,
}: {
  route: DeskCaseRoute;
  apiKey: string;
  /** Told when Google will not draw the map with this key, so the sketch is drawn instead. */
  onUnavailable: () => void;
}) {
  const t = useT();
  const unavailable = useLatestCallback(onUnavailable);
  // Google reports a key it refuses (invalid, restricted, out of quota) by
  // calling this global, not by failing the script.
  useEffect(() => {
    const scope = window as Window & { gm_authFailure?: () => void };
    const earlier = scope.gm_authFailure;
    scope.gm_authFailure = () => {
      earlier?.();
      unavailable();
    };
    return () => {
      scope.gm_authFailure = earlier;
    };
  }, [unavailable]);
  const { mapId, colorScheme } = useMapStyle();
  const stops = useMemo(
    () =>
      route.points.map((point) => ({
        label: point.label,
        state: point.state,
        at: { lat: point.latitude, lng: point.longitude },
      })),
    [route.points],
  );
  const truck: LatLng | null = route.truck
    ? { lat: route.truck.latitude, lng: route.truck.longitude }
    : null;
  const lastDone = stops.reduce((last, stop, index) => (stop.state === "Done" ? index : last), -1);
  const whole = stops.filter((stop) => stop.state !== "Proposed").map((stop) => stop.at);
  const driven = stops.slice(0, lastDone + 1).map((stop) => stop.at);
  if (truck && lastDone >= 0 && lastDone < stops.length - 1) {
    driven.push(truck);
  }
  const proposedAt = stops.findIndex((stop) => stop.state === "Proposed");
  const detour =
    proposedAt > 0 && proposedAt < stops.length - 1
      ? [stops[proposedAt - 1]!.at, stops[proposedAt]!.at, stops[proposedAt + 1]!.at]
      : [];
  const extent = truck ? [...stops.map((stop) => stop.at), truck] : stops.map((stop) => stop.at);

  return (
    <div className="dk-cz-map" role="img" aria-label={t("The route on a map")}>
      <APIProvider apiKey={apiKey} onError={unavailable}>
        <Map
          mapId={mapId}
          colorScheme={colorScheme}
          defaultCenter={US_CENTER}
          defaultZoom={DEFAULT_ZOOM}
          gestureHandling="cooperative"
          disableDefaultUI
        >
          <MapHealth onUnavailable={unavailable} />
          <FitBounds points={extent} />
          <MapPolyline path={whole} colorVar="--muted-foreground" weight={3} opacity={0.55} />
          {driven.length > 1 && (
            <MapPolyline
              path={driven}
              colorVar="--foreground"
              weight={4}
              opacity={0.95}
              zIndex={6}
            />
          )}
          {detour.length > 0 && (
            <MapPolyline
              path={detour}
              colorVar="--info"
              weight={3}
              opacity={0.9}
              dashed
              zIndex={7}
            />
          )}
          {stops.map((stop, index) => (
            <AdvancedMarker
              // oxlint-disable-next-line react/no-array-index-key -- a route's stops keep their order
              key={index}
              position={stop.at}
              title={stop.label}
            >
              <span
                className={cn(
                  "dk-cz-map-stop",
                  stop.state === "Done" && "dk-done",
                  stop.state === "Proposed" && "dk-new",
                )}
              />
            </AdvancedMarker>
          ))}
          {truck && (
            <AdvancedMarker position={truck} title={t("Where the truck last reported")} zIndex={10}>
              <span className="dk-cz-map-truck" />
            </AdvancedMarker>
          )}
        </Map>
      </APIProvider>
      {route.callout !== "" && (
        <span
          className={cn(
            "dk-cz-map-note",
            route.calloutTone === "Warning" && "dk-warn",
            route.calloutTone === "Bad" && "dk-bad",
          )}
        >
          {route.callout}
        </span>
      )}
    </div>
  );
}

/** How long a map is given to draw its first tiles before the sketch takes its place. */
const DRAW_WITHIN_MS = 8000;

/**
 * Tells the exhibit when Google will not draw this map: it cannot make
 * markers, or the map has painted nothing in the time a map takes to load,
 * which is how a refused or exhausted key shows when Google does not say so.
 * The route is then drawn as the sketch rather than as an empty map.
 */
function MapHealth({ onUnavailable }: { onUnavailable: () => void }) {
  const usable = useMarkersUsable();
  const mapDrawn = useMapDrawn();
  useEffect(() => {
    if (usable === false) {
      onUnavailable();
    }
  }, [usable, onUnavailable]);
  useEffect(() => {
    if (mapDrawn) {
      return;
    }
    const timer = window.setTimeout(onUnavailable, DRAW_WITHIN_MS);

    return () => window.clearTimeout(timer);
  }, [mapDrawn, onUnavailable]);

  return null;
}
