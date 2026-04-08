import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoice = createQueryKeys("invoice", {
  get: (invoiceId: string) => ({
    queryKey: ["get", invoiceId],
    queryFn: async () => apiService.invoiceService.getById(invoiceId),
  }),
});
