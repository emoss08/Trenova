import { invoiceShareService } from "@/services/invoice-share";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoiceShare = createQueryKeys("invoice-share", {
  list: (invoiceId: string) => ({
    queryKey: [invoiceId],
    queryFn: async () => invoiceShareService.list(invoiceId),
  }),
  candidates: (invoiceId: string, query: string) => ({
    queryKey: [invoiceId, query],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      invoiceShareService.candidates(invoiceId, query, signal),
  }),
});
