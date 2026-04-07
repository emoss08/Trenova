import { DEFAULT_ZOOM, MAP_ID, US_CENTER } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { cn, formatLocation } from "@/lib/utils";
import type { ShipmentMove, Stop } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";
import { useMemo, useState } from "react";
import { DirectionsPolyline, FitBounds } from "../map-components";

type Point = {
  lat: number;
  lng: number;
  stop: Stop;
};

export function ShipmentRouteMap({
  moves,
  containerClassName,
}: {
  moves: ShipmentMove[];
  containerClassName?: string;
}) {
  const { data: apiKeyData } = useQuery({ ...queries.googleMaps.getAPIKey() });
  const stops = useMemo(() => {
    return moves
      .flatMap((m) => m.stops)
      .sort((a, b) => a.sequence - b.sequence)
      .map((stop) => {
        const lat = Number(stop.location?.latitude);
        const lng = Number(stop.location?.longitude);
        if (Number.isNaN(lat) || Number.isNaN(lng)) return null;
        return { lat, lng, stop } as Point;
      })
      .filter(Boolean) as Point[];
  }, [moves]);

  const coordinates = useMemo(
    () => stops.map((s) => ({ lat: s.lat, lng: s.lng })),
    [stops],
  );

  const center = useMemo(() => {
    if (coordinates.length === 0) return undefined;
    const avgLat =
      coordinates.reduce((s, p) => s + p.lat, 0) / coordinates.length;
    const avgLng =
      coordinates.reduce((s, p) => s + p.lng, 0) / coordinates.length;
    return { lat: avgLat, lng: avgLng };
  }, [coordinates]);

  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const [pinnedIdx, setPinnedIdx] = useState<number | null>(null);
  const activeIdx = pinnedIdx ?? hoveredIdx;

  if (!apiKeyData?.apiKey || !center || coordinates.length === 0) {
    return null;
  }

  return (
    <Container className={containerClassName}>
      <APIProvider apiKey={apiKeyData.apiKey}>
        <Map
          mapId={MAP_ID}
          defaultCenter={US_CENTER}
          defaultZoom={DEFAULT_ZOOM}
          gestureHandling="greedy"
          disableDefaultUI
        >
          <FitBounds points={coordinates} />
          <DirectionsPolyline points={coordinates} />
          {stops.map((pt, idx) => {
            return (
              <AdvancedMarker
                key={`${pt.lat}-${pt.lng}-${idx}`}
                position={{ lat: pt.lat, lng: pt.lng }}
                title={pt.stop.location?.name}
                onMouseEnter={() => setHoveredIdx(idx)}
                onMouseLeave={() =>
                  setHoveredIdx((cur) =>
                    cur === idx && pinnedIdx == null ? null : cur,
                  )
                }
                onClick={() => setPinnedIdx(idx)}
              >
                <StopMarker point={pt} activeIdx={activeIdx} idx={idx} />
              </AdvancedMarker>
            );
          })}
        </Map>
      </APIProvider>
    </Container>
  );
}

function StopMarker({
  point,
  activeIdx,
  idx,
}: {
  point: Point;
  activeIdx: number | null;
  idx: number;
}) {
  const color = (stopType: Stop["type"]) => {
    switch (stopType) {
      case "Pickup":
        return "bg-green-600";
      case "Delivery":
        return "bg-red-600";
      case "SplitPickup":
        return "bg-blue-600";
      case "SplitDelivery":
        return "bg-yellow-600";
      default:
        return "bg-gray-600";
    }
  };
  return (
    <div className="relative">
      <div
        className={cn(
          "size-2.5 rounded-full ring-2 ring-white",
          color(point.stop.type),
        )}
      />
      {activeIdx === idx && (
        <div className="absolute -top-1.5 left-1/2 z-1 -translate-x-1/2 -translate-y-full rounded border border-border bg-popover px-2 py-1 text-2xs whitespace-nowrap text-popover-foreground shadow-md">
          <div className="max-w-[220px] truncate font-medium">
            {point.stop.location?.name || "Location"}
          </div>
          <div className="max-w-[220px] truncate text-muted-foreground">
            {point.stop.location
              ? formatLocation(point.stop.location)
              : `${point.lat.toFixed(4)}, ${point.lng.toFixed(4)}`}
          </div>
        </div>
      )}
    </div>
  );
}

function Container({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn("h-32 w-full overflow-hidden rounded-md border", className)}
    >
      {children}
    </div>
  );
}
