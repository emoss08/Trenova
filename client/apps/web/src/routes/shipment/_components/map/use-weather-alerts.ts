import { weatherAlert } from "@/lib/queries/weather-alert";
import type { WeatherAlertFeature } from "@/types/weather-alert";
import { useQuery } from "@tanstack/react-query";

const ALERTS_REFRESH_MS = 2 * 60 * 1000;

/** Public weather alerts the map can draw: those with an id and an area. */
export function useWeatherAlerts(enabled: boolean): WeatherAlertFeature[] {
  const { data } = useQuery({
    ...weatherAlert.alerts(),
    staleTime: ALERTS_REFRESH_MS,
    refetchInterval: ALERTS_REFRESH_MS,
    enabled,
  });
  if (!enabled) return [];
  return (data?.features ?? []).filter((f) => f.geometry != null && f.properties.id != null);
}
