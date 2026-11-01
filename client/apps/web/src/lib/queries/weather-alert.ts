import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const weatherAlert = createQueryKeys("weatherAlert", {
  alerts: () => ({
    queryKey: ["alerts"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.weatherAlertService.getAlerts({ signal }),
  }),
  detail: (alertId: string) => ({
    queryKey: [alertId],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.weatherAlertService.getAlertDetail(alertId, { signal }),
  }),
});
