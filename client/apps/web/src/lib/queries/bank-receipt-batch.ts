import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceiptBatch = createQueryKeys("bankReceiptBatch", {
  list: () => ({
    queryKey: ["list"],
    queryFn: async ({ signal }) => apiService.bankReceiptBatchService.list({ signal }),
  }),
  get: (batchId: string) => ({
    queryKey: ["get", batchId],
    queryFn: async ({ signal }) => apiService.bankReceiptBatchService.getById(batchId, { signal }),
  }),
});
