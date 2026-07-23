import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoiceAdjustment = createQueryKeys("invoice-adjustment", {
  get: (adjustmentId: string) => ({
    queryKey: ["get", adjustmentId],
    queryFn: async () => apiService.invoiceAdjustmentService.getById(adjustmentId),
  }),
  lineage: (correctionGroupId: string) => ({
    queryKey: ["lineage", correctionGroupId],
    queryFn: async () => apiService.invoiceAdjustmentService.getLineageByGroup(correctionGroupId),
  }),
  batch: (batchId: string) => ({
    queryKey: ["batch", batchId],
    queryFn: async () => apiService.invoiceAdjustmentService.getBatch(batchId),
  }),
  approvals: (params: Record<string, string>) => ({
    queryKey: ["approvals", params],
    queryFn: async () => apiService.invoiceAdjustmentService.listApprovals(params),
  }),
  reconciliation: (params: Record<string, string>) => ({
    queryKey: ["reconciliation", params],
    queryFn: async () => apiService.invoiceAdjustmentService.listReconciliationExceptions(params),
  }),
  batches: (params: Record<string, string>) => ({
    queryKey: ["batches", params],
    queryFn: async () => apiService.invoiceAdjustmentService.listBatches(params),
  }),
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () => apiService.invoiceAdjustmentService.getSummary(),
  }),
});
