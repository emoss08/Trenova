import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const platformBilling = createQueryKeys("platformBilling", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () => apiService.platformBillingService.getSummary(),
  }),
});
