import type { IconComponent } from "@trenova/shared/components/icons";

export type RainViewerFrame = { time: number; path: string };

export type RainViewerData = {
  host: string;
  radar: { past: RainViewerFrame[] };
};

export const OVERLAY_IDS = [
  "vehicles",
  "geofences",
  "addresses",
  "traffic",
  "weather",
  "alerts",
] as const;

export type OverlayId = (typeof OVERLAY_IDS)[number];

export type MapStyleId = "roadmap" | "satellite" | "hybrid" | "terrain";

/**
 * Which routes the live map draws: only the shipments in focus (hovered,
 * selected, or touched by the selected alert), one arc per lane, or none.
 */
export const ROUTE_MODES = ["selected", "lanes", "off"] as const;
export type RouteMode = (typeof ROUTE_MODES)[number];

export const WEATHER_LAYER_OPTIONS = [
  "none",
  "precipitation",
  "wind",
  "temperature",
  "clouds",
  "pressure",
] as const;

export type WeatherLayerId = (typeof WEATHER_LAYER_OPTIONS)[number];
export type WeatherOption = {
  id: WeatherLayerId;
  label: string;
  description: string;
  icon: IconComponent;
};
