import { bankReceiptService } from "@/services/bank-receipt";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceipt = createQueryKeys("bankReceipt", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () => bankReceiptService.getSummary(),
  }),
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => bankReceiptService.getById(id),
  }),
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }: { signal: AbortSignal }) =>
      bankReceiptService.getCard(id, { signal }),
  }),
  suggestions: (id: string) => ({
    queryKey: ["suggestions", id],
    queryFn: async () => bankReceiptService.getSuggestions(id),
  }),
});
