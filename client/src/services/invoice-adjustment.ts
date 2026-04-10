import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  createDraftInvoiceAdjustmentRequestSchema,
  invoiceAdjustmentBatchQueueItemSchema,
  invoiceAdjustmentBatchSchema,
  invoiceAdjustmentLineageSchema,
  invoiceAdjustmentOperationsSummarySchema,
  invoiceAdjustmentPreviewSchema,
  invoiceAdjustmentRequestSchema,
  invoiceAdjustmentSchema,
  invoiceApprovalQueueItemSchema,
  invoiceReconciliationQueueItemSchema,
  updateDraftInvoiceAdjustmentRequestSchema,
  type CreateDraftInvoiceAdjustmentRequest,
  type InvoiceAdjustment,
  type InvoiceAdjustmentBatch,
  type InvoiceAdjustmentBatchQueueItem,
  type InvoiceAdjustmentLineage,
  type InvoiceAdjustmentOperationsSummary,
  type InvoiceAdjustmentPreview,
  type InvoiceAdjustmentRequest,
  type InvoiceApprovalQueueItem,
  type InvoiceReconciliationQueueItem,
  type UpdateDraftInvoiceAdjustmentRequest,
} from "@/types/invoice-adjustment";
import { createLimitOffsetResponse } from "@/types/server";

const invoiceApprovalQueueListSchema = createLimitOffsetResponse(invoiceApprovalQueueItemSchema);
const invoiceReconciliationQueueListSchema = createLimitOffsetResponse(
  invoiceReconciliationQueueItemSchema,
);
const invoiceAdjustmentBatchQueueListSchema = createLimitOffsetResponse(
  invoiceAdjustmentBatchQueueItemSchema,
);

export class InvoiceAdjustmentService {
  public async createDraft(data: CreateDraftInvoiceAdjustmentRequest) {
    const payload = createDraftInvoiceAdjustmentRequestSchema.parse(data);
    const response = await api.post<InvoiceAdjustment>(
      "/billing/invoice-adjustments/drafts/",
      payload,
    );
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async updateDraft(id: InvoiceAdjustment["id"], data: UpdateDraftInvoiceAdjustmentRequest) {
    const payload = updateDraftInvoiceAdjustmentRequestSchema.parse(data);
    const response = await api.patch<InvoiceAdjustment>(
      `/billing/invoice-adjustments/drafts/${id}/`,
      payload,
    );
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async previewDraft(id: InvoiceAdjustment["id"]) {
    const response = await api.post<InvoiceAdjustmentPreview>(
      `/billing/invoice-adjustments/drafts/${id}/preview/`,
      {},
    );
    return safeParse(invoiceAdjustmentPreviewSchema, response, "InvoiceAdjustmentPreview");
  }

  public async submitDraft(id: InvoiceAdjustment["id"]) {
    const response = await api.post<InvoiceAdjustment>(
      `/billing/invoice-adjustments/drafts/${id}/submit/`,
      {},
    );
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async preview(data: InvoiceAdjustmentRequest) {
    const payload = invoiceAdjustmentRequestSchema.parse(data);
    const response = await api.post<InvoiceAdjustmentPreview>(
      "/billing/invoice-adjustments/preview/",
      payload,
    );
    return safeParse(invoiceAdjustmentPreviewSchema, response, "InvoiceAdjustmentPreview");
  }

  public async submit(data: InvoiceAdjustmentRequest) {
    const payload = invoiceAdjustmentRequestSchema.parse(data);
    const response = await api.post<InvoiceAdjustment>(
      "/billing/invoice-adjustments/submit/",
      payload,
    );
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async getById(id: InvoiceAdjustment["id"]) {
    const response = await api.get<InvoiceAdjustment>(`/billing/invoice-adjustments/${id}/`);
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async approve(id: InvoiceAdjustment["id"]) {
    const response = await api.post<InvoiceAdjustment>(
      `/billing/invoice-adjustments/${id}/approve/`,
      {},
    );
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async reject(id: InvoiceAdjustment["id"], reason: InvoiceAdjustment["rejectionReason"]) {
    const response = await api.post<InvoiceAdjustment>(
      `/billing/invoice-adjustments/${id}/reject/`,
      {
        reason,
      },
    );
    return safeParse(invoiceAdjustmentSchema, response, "InvoiceAdjustment");
  }

  public async getLineageByGroup(id: string) {
    const response = await api.get<InvoiceAdjustmentLineage>(
      `/billing/invoice-adjustments/correction-groups/${id}/`,
    );
    return safeParse(invoiceAdjustmentLineageSchema, response, "InvoiceAdjustmentLineage");
  }

  public async getBatch(id: string) {
    const response = await api.get<InvoiceAdjustmentBatch>(
      `/billing/invoice-adjustments/batches/${id}/`,
    );
    return safeParse(invoiceAdjustmentBatchSchema, response, "InvoiceAdjustmentBatch");
  }

  public async listApprovals(params?: Record<string, string>) {
    const endpoint = params
      ? `/billing/invoice-adjustments/approvals/?${new URLSearchParams(params).toString()}`
      : "/billing/invoice-adjustments/approvals/";
    const response = await api.get<{ results: InvoiceApprovalQueueItem[]; count: number }>(
      endpoint,
    );
    return safeParse(invoiceApprovalQueueListSchema, response, "InvoiceApprovalQueueList");
  }

  public async listReconciliationExceptions(params?: Record<string, string>) {
    const endpoint = params
      ? `/billing/invoice-adjustments/reconciliation-exceptions/?${new URLSearchParams(params).toString()}`
      : "/billing/invoice-adjustments/reconciliation-exceptions/";
    const response = await api.get<{ results: InvoiceReconciliationQueueItem[]; count: number }>(
      endpoint,
    );
    return safeParse(
      invoiceReconciliationQueueListSchema,
      response,
      "InvoiceReconciliationExceptionList",
    );
  }

  public async listBatches(params?: Record<string, string>) {
    const endpoint = params
      ? `/billing/invoice-adjustments/batches/?${new URLSearchParams(params).toString()}`
      : "/billing/invoice-adjustments/batches/";
    const response = await api.get<{ results: InvoiceAdjustmentBatchQueueItem[]; count: number }>(
      endpoint,
    );
    return safeParse(invoiceAdjustmentBatchQueueListSchema, response, "InvoiceAdjustmentBatchList");
  }

  public async getSummary() {
    const response = await api.get<InvoiceAdjustmentOperationsSummary>(
      "/billing/invoice-adjustments/summary/",
    );
    return safeParse(
      invoiceAdjustmentOperationsSummarySchema,
      response,
      "InvoiceAdjustmentOperationsSummary",
    );
  }
}
