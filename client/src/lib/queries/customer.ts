import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const customer = createQueryKeys("customer", {
  getBillingProfile: (customerId: string) => ({
    queryKey: ["getBillingProfile", customerId],
    queryFn: async () => apiService.customerService.getBillingProfile(customerId),
  }),
});
