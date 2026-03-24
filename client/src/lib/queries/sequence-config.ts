import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const sequenceConfig = createQueryKeys("sequenceConfig", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.sequenceConfigService.get(),
  }),
});
