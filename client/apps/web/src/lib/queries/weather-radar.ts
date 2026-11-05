import { weatherRadarService } from "@/services/weather-radar";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const weatherRadar = createQueryKeys("weatherRadar", {
  weatherMaps: () => ({
    queryKey: ["weatherMaps"],
    queryFn: async () => weatherRadarService.getWeatherMaps(),
  }),
});
