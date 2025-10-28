"use no memo";
import { queries } from "@/lib/queries";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { StopSchema, StopType } from "@/lib/schemas/stop-schema";
import { cn, formatLocation } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";
import { useMemo, useState } from "react";
import { DirectionsPolyline, FitBounds } from "../map-components";

type Point = {
  lat: number;
  lng: number;
  stop: StopSchema;
};

export function ShipmentRouteMap({
  moves,
}: {
  moves: ShipmentSchema["moves"];
}) {
  const { data: apiKeyData } = useQuery(queries.googleMaps.getAPIKey());
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
      .filter(Boolean) as { lat: number; lng: number; stop: StopSchema }[];
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
    <Container>
      <APIProvider apiKey={apiKeyData.apiKey}>
        <Map
          defaultCenter={center}
          defaultZoom={6}
          mapId="DEMO_MAP_ID"
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
  const color = (stopType: StopSchema["type"]) => {
    switch (stopType) {
      case StopType.enum.Pickup:
        return "bg-green-600";
      case StopType.enum.Delivery:
        return "bg-red-600";
      case StopType.enum.SplitPickup:
        return "bg-blue-600";
      case StopType.enum.SplitDelivery:
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
        <div className="absolute -top-1.5 left-1/2 -translate-x-1/2 -translate-y-full z-1 whitespace-nowrap rounded border border-border bg-popover px-2 py-1 text-2xs text-popover-foreground shadow-md">
          <div className="font-medium truncate max-w-[220px]">
            {point.stop.location?.name || "Location"}
          </div>
          <div className="text-muted-foreground truncate max-w-[220px]">
            {point.stop.location
              ? formatLocation(point.stop.location)
              : `${point.lat.toFixed(4)}, ${point.lng.toFixed(4)}`}
          </div>
        </div>
      )}
    </div>
  );
}

function Container({ children }: { children: React.ReactNode }) {
  return (
    <div className="h-40 w-full overflow-hidden rounded-md border">
      {children}
    </div>
  );
}
