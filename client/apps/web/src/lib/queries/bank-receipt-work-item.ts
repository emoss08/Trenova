import { bankReceiptWorkItemService } from "@/services/bank-receipt-work-item";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceiptWorkItem = createQueryKeys("bankReceiptWorkItem", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => bankReceiptWorkItemService.getById(id),
  }),
});
