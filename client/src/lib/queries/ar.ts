import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const ar = createQueryKeys("ar", {
  aging: (params?: Record<string, string>) => ({
    queryKey: ["aging", params],
    queryFn: async () => apiService.arService.getAging(params),
  }),
  customerLedger: (customerId: string, params?: Record<string, string>) => ({
    queryKey: ["customerLedger", customerId, params],
    queryFn: async () => apiService.arService.getCustomerLedger(customerId, params),
  }),
  openItems: (params?: Record<string, string>) => ({
    queryKey: ["openItems", params],
    queryFn: async () => apiService.arService.getOpenItems(params),
  }),
  customerStatement: (customerId: string, params?: Record<string, string>) => ({
    queryKey: ["customerStatement", customerId, params],
    queryFn: async () => apiService.arService.getCustomerStatement(customerId, params),
  }),
});
