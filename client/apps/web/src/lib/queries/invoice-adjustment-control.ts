import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoiceAdjustmentControl = createQueryKeys("invoiceAdjustmentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      apiService.invoiceAdjustmentControlService.get({ signal }),
  }),
});
