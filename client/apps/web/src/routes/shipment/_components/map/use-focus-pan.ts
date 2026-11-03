import type { WeatherAlertFeature } from "@/types/weather-alert";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useMap } from "@vis.gl/react-google-maps";
import { useEffect } from "react";
import { useMapFocusStore } from "./map-focus-store";
import type { MapShipment } from "./map-shipment-model";
import type { Latlng } from "./shipment-route-coordinates";
import { alertBounds, type AlertMatches } from "./weather-alert-matching";

const PAN_DELAY_MS = 120;
const VIEWPORT_MARGIN_RATIO = 0.1;
/** Wider than this, the detail card sits on the right; narrower, it covers the top. */
const SIDE_CARD_MIN_WIDTH_PX = 640;
const SIDE_CARD_PADDING = { top: 56, right: 352, bottom: 56, left: 56 };
const TOP_CARD_PADDING = { top: 260, right: 32, bottom: 32, left: 32 };

/**
 * Brings what was just selected into view: a shipment is panned to when its
 * mark is off screen or under the edge, and an alert is fitted together with
 * the shipments it touches, clear of the detail card.
 */
export function FocusAutoPan({
  mapInstanceId,
  models,
  features,
  matches,
}: {
  mapInstanceId: string;
  models: readonly MapShipment[];
  features: readonly WeatherAlertFeature[];
  matches: AlertMatches;
}) {
  const map = useMap(mapInstanceId);
  const focusKind = useMapFocusStore((s) => s.focus?.kind ?? null);
  const focusId = useMapFocusStore((s) => s.focus?.id ?? null);

  // Data refreshes do not move the map; only a new selection does.
  const bringIntoView = useLatestCallback(() => {
    if (!map || !focusId) return;
    if (focusKind === "shipment") {
      const model = models.find((m) => m.id === focusId);
      if (model && !isPointWithinInsetBounds(map, model.position)) map.panTo(model.position);
      return;
    }
    const feature = features.find((f) => f.properties.id === focusId);
    const bounds = feature ? alertBounds(feature) : null;
    if (!bounds) return;
    const affected = new Set((matches.byAlert.get(focusId) ?? []).map((hit) => hit.shipmentId));
    for (const model of models) {
      if (!affected.has(model.id)) continue;
      bounds.south = Math.min(bounds.south, model.position.lat);
      bounds.north = Math.max(bounds.north, model.position.lat);
      bounds.west = Math.min(bounds.west, model.position.lng);
      bounds.east = Math.max(bounds.east, model.position.lng);
    }
    const sideCard = (map.getDiv().clientWidth ?? 0) >= SIDE_CARD_MIN_WIDTH_PX;
    map.fitBounds(bounds, sideCard ? SIDE_CARD_PADDING : TOP_CARD_PADDING);
  });

  useEffect(() => {
    if (!map || !focusId) return;
    const timeout = window.setTimeout(bringIntoView, PAN_DELAY_MS);
    return () => window.clearTimeout(timeout);
  }, [map, focusKind, focusId, bringIntoView]);

  return null;
}

function isPointWithinInsetBounds(map: google.maps.Map, point: Latlng) {
  const bounds = map.getBounds();
  if (!bounds) return true;

  const json = bounds.toJSON();
  const latMargin = (json.north - json.south) * VIEWPORT_MARGIN_RATIO;
  const lngMargin = (json.east - json.west) * VIEWPORT_MARGIN_RATIO;

  return (
    point.lat >= json.south + latMargin &&
    point.lat <= json.north - latMargin &&
    point.lng >= json.west + lngMargin &&
    point.lng <= json.east - lngMargin
  );
}
