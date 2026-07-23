import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const location = createQueryKeys("location", {
  geofences: (limit = 100) => ({
    queryKey: ["geofences", limit],
    queryFn: async () => apiService.locationService.listGeofenced(limit),
  }),
});
