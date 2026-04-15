import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const weatherRadar = createQueryKeys("weatherRadar", {
  weatherMaps: () => ({
    queryKey: ["weatherMaps"],
    queryFn: async () => apiService.weatherRadarService.getWeatherMaps(),
  }),
});
