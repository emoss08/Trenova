import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const weatherAlert = createQueryKeys("weatherAlert", {
  alerts: () => ({
    queryKey: ["alerts"],
    queryFn: async () => apiService.weatherAlertService.getAlerts(),
  }),
  detail: (alertId: string) => ({
    queryKey: [alertId],
    queryFn: async () => apiService.weatherAlertService.getAlertDetail(alertId),
  }),
});
