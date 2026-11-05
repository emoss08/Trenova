import { weatherAlertService } from "@/services/weather-alert";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const weatherAlert = createQueryKeys("weatherAlert", {
  alerts: () => ({
    queryKey: ["alerts"],
    queryFn: async () => weatherAlertService.getAlerts(),
  }),
  detail: (alertId: string) => ({
    queryKey: [alertId],
    queryFn: async () => weatherAlertService.getAlertDetail(alertId),
  }),
});
