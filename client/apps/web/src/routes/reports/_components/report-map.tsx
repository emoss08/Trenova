import { queries } from "@/lib/queries";
import type { ReportChartSpec } from "@/types/report";
import { useQuery } from "@tanstack/react-query";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";
import { useMapId } from "@/hooks/use-map-id";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";

type ReportMapProps = {
  chart: ReportChartSpec;
  columns: { id: string; label: string }[];
  rows: unknown[][];
  className?: string;
};

type MapPoint = {
  key: string;
  lat: number;
  lng: number;
  label: string;
};

/** Rough centre of the continental US, for a map with nothing to show. */
const FALLBACK_CENTER = { lat: 39.5, lng: -98.35 };

/**
 * Coordinates arrive as whatever the executor decoded — a decimal string, a
 * number, or null for a row that has no location yet. Anything that is not a
 * finite number in range is dropped rather than plotted at (0, 0), which is in
 * the Gulf of Guinea and looks like real data.
 */
function toCoordinate(value: unknown, limit: number): number | null {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric) || numeric === 0) return null;
  return Math.abs(numeric) <= limit ? numeric : null;
}

/**
 * Marker titles come from an arbitrary result column, so only values that
 * already read as text are used — an object would stringify to
 * "[object Object]" and label every pin identically.
 */
function pinLabel(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

export function ReportMap({ chart, columns, rows, className }: ReportMapProps) {
  const mapId = useMapId();
  const { data: googleMaps } = useQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
  });

  const points = useMemo<MapPoint[]>(() => {
    const indexOf = (id: string | undefined) =>
      id ? columns.findIndex((column) => column.id === id) : -1;

    const latIndex = indexOf(chart.latColumnId);
    const lngIndex = indexOf(chart.lngColumnId);
    if (latIndex < 0 || lngIndex < 0) return [];

    const labelIndex = indexOf(chart.labelColumnId);
    const collected: MapPoint[] = [];

    rows.forEach((row, rowIndex) => {
      const lat = toCoordinate(row[latIndex], 90);
      const lng = toCoordinate(row[lngIndex], 180);
      if (lat === null || lng === null) return;

      collected.push({
        key: `${rowIndex}:${lat},${lng}`,
        lat,
        lng,
        label: labelIndex >= 0 ? pinLabel(row[labelIndex]) : "",
      });
    });

    return collected;
  }, [chart.latColumnId, chart.lngColumnId, chart.labelColumnId, columns, rows]);

  // Centring on the mean keeps every pin roughly in frame without needing the
  // map instance, which is what lets this render identically in a tile and in
  // the builder preview.
  const center = useMemo(() => {
    if (points.length === 0) return FALLBACK_CENTER;
    const sum = points.reduce(
      (acc, point) => ({ lat: acc.lat + point.lat, lng: acc.lng + point.lng }),
      { lat: 0, lng: 0 },
    );
    return { lat: sum.lat / points.length, lng: sum.lng / points.length };
  }, [points]);

  if (!chart.latColumnId || !chart.lngColumnId) {
    return <MapNotice message="Choose a latitude and a longitude column." />;
  }
  if (!googleMaps?.config.apiKey) {
    return <MapNotice message="Maps are unavailable — no Google Maps key is configured." />;
  }
  if (points.length === 0) {
    return <MapNotice message="This report returned no rows with usable coordinates." />;
  }

  return (
    <div className={cn("relative h-full w-full overflow-hidden rounded-md", className)}>
      <div className="absolute inset-0">
        <APIProvider apiKey={googleMaps.config.apiKey}>
          <Map
            defaultCenter={center}
            defaultZoom={points.length === 1 ? 9 : 4}
            mapId={mapId}
            gestureHandling="cooperative"
            disableDefaultUI
          >
            {points.map((point) => (
              <AdvancedMarker
                key={point.key}
                position={{ lat: point.lat, lng: point.lng }}
                title={point.label}
              />
            ))}
          </Map>
        </APIProvider>
      </div>
    </div>
  );
}

function MapNotice({ message }: { message: string }) {
  return (
    <div className="flex h-full w-full items-center justify-center p-4 text-center text-xs text-muted-foreground">
      {message}
    </div>
  );
}
