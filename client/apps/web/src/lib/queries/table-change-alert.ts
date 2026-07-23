import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableChangeAlert = createQueryKeys("tableChangeAlert", {
  subscriptions: (params?: { limit?: number; offset?: number; query?: string }) => ({
    queryKey: [params],
    queryFn: async () => apiService.tableChangeAlertService.listSubscriptions(params),
  }),
  allowlistedTables: () => ({
    queryKey: ["allowlisted-tables"],
    queryFn: async () => apiService.tableChangeAlertService.listAllowlistedTables(),
  }),
});
