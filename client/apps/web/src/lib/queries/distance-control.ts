import { distanceControlService } from "@/services/distance-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const distanceControl = createQueryKeys("distanceControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => distanceControlService.get(),
  }),
});
