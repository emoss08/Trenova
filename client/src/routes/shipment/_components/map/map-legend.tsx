import type { OverlayId, WeatherLayerId } from "@/types/shipment-map";
import { ControlPosition, MapControl } from "@vis.gl/react-google-maps";
import { useMemo } from "react";

type LegendEntry = {
  key: string;
  swatch: React.ReactNode;
  label: string;
};

function RouteSwatch() {
  return (
    <svg width="20" height="6" className="shrink-0">
      <line x1="0" y1="3" x2="20" y2="3" stroke="#3b82f6" strokeWidth="2" strokeDasharray="4 3" />
    </svg>
  );
}

function DotSwatch({ color }: { color: string }) {
  return (
    <span
      className="inline-block size-2.5 shrink-0 rounded-full border"
      style={{ backgroundColor: color, borderColor: color }}
    />
  );
}

function RingSwatch({ color }: { color: string }) {
  return (
    <span
      className="inline-block size-2.5 shrink-0 rounded-full border-2"
      style={{ borderColor: color, backgroundColor: `${color}20` }}
    />
  );
}

function GradientSwatch({ from, to }: { from: string; to: string }) {
  return (
    <span
      className="inline-block h-2 w-5 shrink-0 rounded-sm"
      style={{
        background: `linear-gradient(to right, ${from}, ${to})`,
      }}
    />
  );
}

const WEATHER_LEGEND: Record<Exclude<WeatherLayerId, "none">, LegendEntry> = {
  precipitation: {
    key: "precipitation",
    swatch: <GradientSwatch from="#a3d9f5" to="#1e3a8a" />,
    label: "Precipitation",
  },
  wind: {
    key: "wind",
    swatch: <GradientSwatch from="#fff" to="#6366f1" />,
    label: "Wind",
  },
  clouds: {
    key: "clouds",
    swatch: <GradientSwatch from="#f1f5f9" to="#64748b" />,
    label: "Clouds",
  },
  temperature: {
    key: "temperature",
    swatch: <GradientSwatch from="#3b82f6" to="#ef4444" />,
    label: "Temp",
  },
  pressure: {
    key: "pressure",
    swatch: <GradientSwatch from="#22c55e" to="#a855f7" />,
    label: "Pressure",
  },
};

export function MapLegend({
  overlays,
  weatherLayer,
}: {
  overlays: Record<OverlayId, boolean>;
  weatherLayer: WeatherLayerId;
}) {
  const entries = useMemo(() => {
    const result: LegendEntry[] = [];

    if (overlays.routes) {
      result.push({ key: "route", swatch: <RouteSwatch />, label: "Route" });
    }
    if (overlays.stops) {
      result.push(
        { key: "pickup", swatch: <DotSwatch color="#3b82f6" />, label: "Pickup" },
        { key: "delivery", swatch: <DotSwatch color="#16a34a" />, label: "Delivery" },
      );
    }
    if (overlays.vehicles) {
      result.push({ key: "vehicle", swatch: <DotSwatch color="#000000" />, label: "Vehicle" });
    }
    if (overlays.geofences) {
      result.push({ key: "geofence", swatch: <RingSwatch color="#3b82f6" />, label: "Geofence" });
    }
    if (overlays.traffic) {
      result.push({
        key: "traffic",
        swatch: <GradientSwatch from="#22c55e" to="#ef4444" />,
        label: "Traffic",
      });
    }
    if (weatherLayer !== "none") {
      result.push(WEATHER_LEGEND[weatherLayer]);
    }

    return result;
  }, [overlays, weatherLayer]);

  if (entries.length === 0) return null;

  return (
    <MapControl position={ControlPosition.LEFT_BOTTOM}>
      <div className="m-2.5 flex items-center gap-3 rounded-lg border bg-background/95 px-2.5 py-1.5 shadow-sm backdrop-blur-sm">
        {entries.map((entry) => (
          <div key={entry.key} className="flex items-center gap-1.5">
            {entry.swatch}
            <span className="text-2xs text-muted-foreground">{entry.label}</span>
          </div>
        ))}
      </div>
    </MapControl>
  );
}
