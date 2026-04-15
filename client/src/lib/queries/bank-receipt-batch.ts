import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceiptBatch = createQueryKeys("bankReceiptBatch", {
  list: () => ({
    queryKey: ["list"],
    queryFn: async () => apiService.bankReceiptBatchService.list(),
  }),
  get: (batchId: string) => ({
    queryKey: ["get", batchId],
    queryFn: async () => apiService.bankReceiptBatchService.getById(batchId),
  }),
});
