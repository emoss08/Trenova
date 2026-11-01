import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceipt = createQueryKeys("bankReceipt", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async ({ signal }) => apiService.bankReceiptService.getSummary({ signal }),
  }),
  exceptions: () => ({
    queryKey: ["exceptions"],
    queryFn: async ({ signal }) => apiService.bankReceiptService.getExceptions({ signal }),
  }),
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async ({ signal }) => apiService.bankReceiptService.getById(id, { signal }),
  }),
  suggestions: (id: string) => ({
    queryKey: ["suggestions", id],
    queryFn: async ({ signal }) => apiService.bankReceiptService.getSuggestions(id, { signal }),
  }),
});
