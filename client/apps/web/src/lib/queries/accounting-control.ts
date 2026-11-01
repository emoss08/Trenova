import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingControl = createQueryKeys("accountingControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.accountingControlService.get({ signal }),
  }),
});
