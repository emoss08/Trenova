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
  suggestions: (id: string) => ({
    queryKey: ["suggestions", id],
    queryFn: async () => bankReceiptService.getSuggestions(id),
  }),
});
