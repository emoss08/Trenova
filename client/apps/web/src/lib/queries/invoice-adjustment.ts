import { apiService } from "@/services/api";
import type { InvoiceAdjustmentQueueParams } from "@/types/invoice-adjustment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

export const invoiceAdjustment = createQueryKeys("invoice-adjustment", {
  get: (adjustmentId: string) => ({
    queryKey: ["get", adjustmentId],
    queryFn: async ({ signal }: QueryContext) =>
      apiService.invoiceAdjustmentService.getById(adjustmentId, { signal }),
  }),
  lineage: (correctionGroupId: string) => ({
    queryKey: ["lineage", correctionGroupId],
    queryFn: async ({ signal }: QueryContext) =>
      apiService.invoiceAdjustmentService.getLineageByGroup(correctionGroupId, { signal }),
  }),
  batch: (batchId: string) => ({
    queryKey: ["batch", batchId],
    queryFn: async ({ signal }: QueryContext) =>
      apiService.invoiceAdjustmentService.getBatch(batchId, { signal }),
  }),
  reconciliation: (params: InvoiceAdjustmentQueueParams) => ({
    queryKey: ["reconciliation", params],
    queryFn: async ({ signal }: QueryContext) =>
      apiService.invoiceAdjustmentService.listReconciliationExceptions(params, { signal }),
  }),
  batches: (params: InvoiceAdjustmentQueueParams) => ({
    queryKey: ["batches", params],
    queryFn: async ({ signal }: QueryContext) =>
      apiService.invoiceAdjustmentService.listBatches(params, { signal }),
  }),
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async ({ signal }: QueryContext) =>
      apiService.invoiceAdjustmentService.getSummary({ signal }),
  }),
});
