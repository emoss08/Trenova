import { invoiceAdjustmentService } from "@/services/invoice-adjustment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoiceAdjustment = createQueryKeys("invoice-adjustment", {
  get: (adjustmentId: string) => ({
    queryKey: ["get", adjustmentId],
    queryFn: async () => invoiceAdjustmentService.getById(adjustmentId),
  }),
  lineage: (correctionGroupId: string) => ({
    queryKey: ["lineage", correctionGroupId],
    queryFn: async () => invoiceAdjustmentService.getLineageByGroup(correctionGroupId),
  }),
  batch: (batchId: string) => ({
    queryKey: ["batch", batchId],
    queryFn: async () => invoiceAdjustmentService.getBatch(batchId),
  }),
  reconciliation: (params: Record<string, string>) => ({
    queryKey: ["reconciliation", params],
    queryFn: async () => invoiceAdjustmentService.listReconciliationExceptions(params),
  }),
  batches: (params: Record<string, string>) => ({
    queryKey: ["batches", params],
    queryFn: async () => invoiceAdjustmentService.listBatches(params),
  }),
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () => invoiceAdjustmentService.getSummary(),
  }),
});
