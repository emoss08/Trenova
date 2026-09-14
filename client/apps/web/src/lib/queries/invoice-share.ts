import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoiceShare = createQueryKeys("invoice-share", {
  list: (invoiceId: string) => ({
    queryKey: [invoiceId],
    queryFn: async () => apiService.invoiceShareService.list(invoiceId),
  }),
  candidates: (invoiceId: string, query: string) => ({
    queryKey: [invoiceId, query],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.invoiceShareService.candidates(invoiceId, query, signal),
  }),
});
