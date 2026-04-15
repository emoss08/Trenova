import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceiptWorkItem = createQueryKeys("bankReceiptWorkItem", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => apiService.bankReceiptWorkItemService.getById(id),
  }),
});
