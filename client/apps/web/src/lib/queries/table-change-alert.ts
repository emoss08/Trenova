import { tableChangeAlertService } from "@/services/table-change-alert";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableChangeAlert = createQueryKeys("tableChangeAlert", {
  subscriptions: (params?: { limit?: number; offset?: number; query?: string }) => ({
    queryKey: [params],
    queryFn: async () => tableChangeAlertService.listSubscriptions(params),
  }),
  allowlistedTables: () => ({
    queryKey: ["allowlisted-tables"],
    queryFn: async () => tableChangeAlertService.listAllowlistedTables(),
  }),
});
