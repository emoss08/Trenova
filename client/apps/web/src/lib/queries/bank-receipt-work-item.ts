import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceiptWorkItem = createQueryKeys("bankReceiptWorkItem", {
  list: () => ({
    queryKey: ["list"],
    queryFn: async ({ signal }) => apiService.bankReceiptWorkItemService.list({ signal }),
  }),
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async ({ signal }) => apiService.bankReceiptWorkItemService.getById(id, { signal }),
  }),
});
