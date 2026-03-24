import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const billingControl = createQueryKeys("billingControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => apiService.billingControlService.get(),
  }),
});
