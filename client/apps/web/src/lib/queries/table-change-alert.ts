import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableChangeAlert = createQueryKeys("tableChangeAlert", {
  allowlistedTables: () => ({
    queryKey: ["allowlisted-tables"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.tableChangeAlertService.listAllowlistedTables({ signal }),
  }),
});
