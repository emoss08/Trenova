import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const distanceControl = createQueryKeys("distanceControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.distanceControlService.get(),
  }),
});

