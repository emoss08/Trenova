import { bankReceiptBatchService } from "@/services/bank-receipt-batch";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bankReceiptBatch = createQueryKeys("bankReceiptBatch", {
  list: () => ({
    queryKey: ["list"],
    queryFn: async () => bankReceiptBatchService.list(),
  }),
  get: (batchId: string) => ({
    queryKey: ["get", batchId],
    queryFn: async () => bankReceiptBatchService.getById(batchId),
  }),
});
