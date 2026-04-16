import type { BadgeVariant } from "@/components/ui/badge";

export type WeatherAlertCategory =
  | "winter_weather"
  | "wind_storm"
  | "flood_water"
  | "fire"
  | "heat"
  | "tornado_severe_storm"
  | "tropical_storm_hurricane"
  | "other";

export type WeatherAlertActivityType = "issued" | "updated" | "expired" | "cancelled";

export type WeatherAlertGeometry = {
  type: string;
  coordinates: number[][][] | number[][][][];
};

export type WeatherAlertProperties = {
  id: string;
  nwsId: string;
  event: string;
  severity?: string;
  urgency?: string;
  certainty?: string;
  headline?: string;
  description?: string;
  instruction?: string;
  areaDesc?: string;
  effective?: number | null;
  expires?: number | null;
  onset?: number | null;
  ends?: number | null;
  status?: string;
  messageType?: string;
  senderName?: string;
  response?: string;
  category?: string;
  alertCategory: WeatherAlertCategory;
  firstSeenAt: number;
  lastUpdatedAt: number;
  expiredAt?: number | null;
};

export type WeatherAlertFeature = {
  type: "Feature";
  id: string;
  geometry: WeatherAlertGeometry;
  properties: WeatherAlertProperties;
};

export type WeatherAlertFeatureCollection = {
  type: "FeatureCollection";
  features: WeatherAlertFeature[];
};

export type WeatherAlertActivity = {
  id: string;
  organizationId: string;
  businessUnitId: string;
  weatherAlertId: string;
  activityType: WeatherAlertActivityType;
  timestamp: number;
  details: Record<string, any> | null;
  createdAt: number;
  updatedAt: number;
};

export type WeatherAlertDetail = {
  feature: WeatherAlertFeature;
  activities: WeatherAlertActivity[];
};

type AlertCategoryConfig = {
  stroke: string;
  fill: string;
  label: string;
};

export const ALERT_CATEGORY_CONFIG: Record<WeatherAlertCategory, AlertCategoryConfig> = {
  winter_weather: { stroke: "#6366f1", fill: "#6366f180", label: "Winter Weather" },
  wind_storm: { stroke: "#ca8a04", fill: "#ca8a0480", label: "Wind & Storm" },
  flood_water: { stroke: "#16a34a", fill: "#16a34a80", label: "Flood & Water" },
  fire: { stroke: "#ea580c", fill: "#ea580c80", label: "Fire" },
  heat: { stroke: "#dc2626", fill: "#dc262680", label: "Heat" },
  tornado_severe_storm: { stroke: "#e11d48", fill: "#e11d4880", label: "Tornado & Severe Storm" },
  tropical_storm_hurricane: {
    stroke: "#7c3aed",
    fill: "#7c3aed80",
    label: "Tropical Storms & Hurricanes",
  },
  other: { stroke: "#a78bfa", fill: "#a78bfa80", label: "Other Alerts" },
};

export const SEVERITY_BADGE_MAP: Record<string, BadgeVariant> = {
  Extreme: "inactive",
  Severe: "orange",
  Moderate: "warning",
  Minor: "info",
};

export const ACTIVITY_TYPE_COLORS: Record<WeatherAlertActivityType, string> = {
  issued: "#16a34a",
  updated: "#3b82f6",
  expired: "#6b7280",
  cancelled: "#dc2626",
};
