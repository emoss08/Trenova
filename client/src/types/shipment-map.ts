import type { LucideIcon } from "lucide-react";

export type RainViewerFrame = { time: number; path: string };

export type RainViewerData = {
  host: string;
  radar: { past: RainViewerFrame[] };
};

export const OVERLAY_IDS = [
  "vehicles",
  "routes",
  "stops",
  "geofences",
  "traffic",
  "weather",
  "alerts",
] as const;

export type OverlayId = (typeof OVERLAY_IDS)[number];

export type MapStyleId = "roadmap" | "satellite" | "hybrid" | "terrain";

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
  icon: LucideIcon;
};
