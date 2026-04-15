import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceipt = createQueryKeys("bankReceipt", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () => apiService.bankReceiptService.getSummary(),
  }),
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => apiService.bankReceiptService.getById(id),
  }),
  suggestions: (id: string) => ({
    queryKey: ["suggestions", id],
    queryFn: async () => apiService.bankReceiptService.getSuggestions(id),
  }),
});
