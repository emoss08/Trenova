import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const billingQueue = createQueryKeys("billingQueue", {
  stats: () => ({
    queryKey: ["stats"],
    queryFn: async () => apiService.billingQueueService.getStats(),
  }),
  get: (itemId: string) => ({
    queryKey: ["get", itemId],
    queryFn: async () =>
      apiService.billingQueueService.getById(itemId, {
        expandShipmentDetails: "true",
      }),
  }),
});
