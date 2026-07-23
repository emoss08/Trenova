import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const shipmentControl = createQueryKeys("shipmentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.shipmentControlService.get(),
  }),
});
