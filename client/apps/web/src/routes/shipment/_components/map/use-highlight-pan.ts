import type { Shipment } from "@/types/shipment";
import { useMap } from "@vis.gl/react-google-maps";
import { useEffect, useMemo } from "react";
import { useCommandCenterStore } from "../command-center/store";
import { getShipmentCurrentLatLng, type Latlng } from "./shipment-route-coordinates";
import { useMapShipments } from "./use-map-shipments";

const HIGHLIGHT_PAN_DELAY_MS = 120;
const VIEWPORT_MARGIN_RATIO = 0.1;

export function useHighlightPan(mapInstanceId: string, enabled = true) {
  const map = useMap(mapInstanceId);
  const highlightId = useCommandCenterStore.use.highlightId();
  const { data } = useMapShipments(enabled);

  const highlightedPoint = useMemo(() => {
    if (!highlightId) return null;

    const shipment = data?.results.find((s) => s.id === highlightId);
    if (!shipment) return null;

    return getShipmentCurrentLatLng(shipment as Shipment);
  }, [data?.results, highlightId]);

  useEffect(() => {
    if (!map || !highlightId || !highlightedPoint) return;

    const timeout = window.setTimeout(() => {
      if (!isPointWithinInsetBounds(map, highlightedPoint)) {
        map.panTo(highlightedPoint);
      }
    }, HIGHLIGHT_PAN_DELAY_MS);

    return () => window.clearTimeout(timeout);
  }, [highlightId, highlightedPoint, map]);
}

export function HighlightAutoPan({
  mapInstanceId,
  enabled = true,
}: {
  mapInstanceId: string;
  enabled?: boolean;
}) {
  useHighlightPan(mapInstanceId, enabled);
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
