import {
  ApproveInvoiceAdjustmentDetailDocument,
  CreateInvoiceAdjustmentDraftDocument,
  InvoiceAdjustmentBatchDocument,
  InvoiceAdjustmentBatchesDocument,
  InvoiceAdjustmentDetailDocument,
  InvoiceAdjustmentDraftPreviewDocument,
  InvoiceAdjustmentLineageDocument,
  InvoiceAdjustmentReconciliationExceptionsDocument,
  InvoiceAdjustmentWorkloadSummaryDocument,
  RejectInvoiceAdjustmentDetailDocument,
  SubmitInvoiceAdjustmentDraftDocument,
  UpdateInvoiceAdjustmentDraftDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  createDraftInvoiceAdjustmentRequestSchema,
  invoiceAdjustmentBatchPageSchema,
  invoiceAdjustmentBatchSchema,
  invoiceAdjustmentLineageSchema,
  invoiceAdjustmentOperationsSummarySchema,
  invoiceAdjustmentPreviewSchema,
  invoiceAdjustmentSchema,
  invoiceReconciliationPageSchema,
  updateDraftInvoiceAdjustmentRequestSchema,
  type CreateDraftInvoiceAdjustmentRequest,
  type InvoiceAdjustment,
  type InvoiceAdjustmentQueueParams,
  type UpdateDraftInvoiceAdjustmentRequest,
} from "@/types/invoice-adjustment";

type RequestOptions = { signal?: AbortSignal };

function queueInput(params: InvoiceAdjustmentQueueParams) {
  return {
    limit: params.limit,
    query: params.query?.trim() || undefined,
    status: params.status || undefined,
  };
}

export class InvoiceAdjustmentService {
  public async createDraft(data: CreateDraftInvoiceAdjustmentRequest) {
    const { invoiceId } = createDraftInvoiceAdjustmentRequestSchema.parse(data);
    const response = await requestGraphQL<{ createInvoiceAdjustmentDraft: unknown }>({
      document: CreateInvoiceAdjustmentDraftDocument,
      operationName: "CreateInvoiceAdjustmentDraft",
      variables: { invoiceId },
    });
    return safeParse(
      invoiceAdjustmentSchema,
      response.createInvoiceAdjustmentDraft,
      "InvoiceAdjustment",
    );
  }

  public async updateDraft(id: InvoiceAdjustment["id"], data: UpdateDraftInvoiceAdjustmentRequest) {
    const input = updateDraftInvoiceAdjustmentRequestSchema.parse(data);
    const response = await requestGraphQL<{ updateInvoiceAdjustmentDraft: unknown }>({
      document: UpdateInvoiceAdjustmentDraftDocument,
      operationName: "UpdateInvoiceAdjustmentDraft",
      variables: {
        id,
        input: {
          kind: input.kind,
          rebillStrategy: input.rebillStrategy,
          reason: input.reason,
          referencedDocumentIds: input.referencedDocumentIds,
          lines: input.lines.map((line) => ({
            originalLineId: line.originalLineId,
            creditQuantity: String(line.creditQuantity ?? 0),
            creditAmount: String(line.creditAmount ?? 0),
            rebillQuantity: String(line.rebillQuantity ?? 0),
            rebillAmount: String(line.rebillAmount ?? 0),
            description: line.description,
            replacementPayload: line.replacementPayload,
          })),
        },
      },
    });
    return safeParse(
      invoiceAdjustmentSchema,
      response.updateInvoiceAdjustmentDraft,
      "InvoiceAdjustment",
    );
  }

  /** What submitting the draft would credit and rebill; nothing is saved. */
  public async previewDraft(id: InvoiceAdjustment["id"], options: RequestOptions = {}) {
    const response = await requestGraphQL<{ invoiceAdjustmentDraftPreview: unknown }>({
      document: InvoiceAdjustmentDraftPreviewDocument,
      operationName: "InvoiceAdjustmentDraftPreview",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(
      invoiceAdjustmentPreviewSchema,
      response.invoiceAdjustmentDraftPreview,
      "InvoiceAdjustmentPreview",
    );
  }

  public async submitDraft(id: InvoiceAdjustment["id"]) {
    const response = await requestGraphQL<{ submitInvoiceAdjustmentDraft: unknown }>({
      document: SubmitInvoiceAdjustmentDraftDocument,
      operationName: "SubmitInvoiceAdjustmentDraft",
      variables: { id },
    });
    return safeParse(
      invoiceAdjustmentSchema,
      response.submitInvoiceAdjustmentDraft,
      "InvoiceAdjustment",
    );
  }

  public async getById(id: InvoiceAdjustment["id"], options: RequestOptions = {}) {
    const response = await requestGraphQL<{ invoiceAdjustment: unknown }>({
      document: InvoiceAdjustmentDetailDocument,
      operationName: "InvoiceAdjustmentDetail",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(invoiceAdjustmentSchema, response.invoiceAdjustment, "InvoiceAdjustment");
  }

  public async approve(id: InvoiceAdjustment["id"]) {
    const response = await requestGraphQL<{ approveInvoiceAdjustment: unknown }>({
      document: ApproveInvoiceAdjustmentDetailDocument,
      operationName: "ApproveInvoiceAdjustmentDetail",
      variables: { adjustmentId: id },
    });
    return safeParse(
      invoiceAdjustmentSchema,
      response.approveInvoiceAdjustment,
      "InvoiceAdjustment",
    );
  }

  public async reject(id: InvoiceAdjustment["id"], reason: InvoiceAdjustment["rejectionReason"]) {
    const response = await requestGraphQL<{ rejectInvoiceAdjustment: unknown }>({
      document: RejectInvoiceAdjustmentDetailDocument,
      operationName: "RejectInvoiceAdjustmentDetail",
      variables: { input: { adjustmentId: id, reason: reason ?? "" } },
    });
    return safeParse(
      invoiceAdjustmentSchema,
      response.rejectInvoiceAdjustment,
      "InvoiceAdjustment",
    );
  }

  public async getLineageByGroup(id: string, options: RequestOptions = {}) {
    const response = await requestGraphQL<{ invoiceAdjustmentLineage: unknown }>({
      document: InvoiceAdjustmentLineageDocument,
      operationName: "InvoiceAdjustmentLineage",
      variables: { correctionGroupId: id },
      signal: options.signal,
    });
    return safeParse(
      invoiceAdjustmentLineageSchema,
      response.invoiceAdjustmentLineage,
      "InvoiceAdjustmentLineage",
    );
  }

  public async getBatch(id: string, options: RequestOptions = {}) {
    const response = await requestGraphQL<{ invoiceAdjustmentBatch: unknown }>({
      document: InvoiceAdjustmentBatchDocument,
      operationName: "InvoiceAdjustmentBatch",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(
      invoiceAdjustmentBatchSchema,
      response.invoiceAdjustmentBatch,
      "InvoiceAdjustmentBatch",
    );
  }

  public async listReconciliationExceptions(
    params: InvoiceAdjustmentQueueParams,
    options: RequestOptions = {},
  ) {
    const response = await requestGraphQL<{ invoiceAdjustmentReconciliationExceptions: unknown }>({
      document: InvoiceAdjustmentReconciliationExceptionsDocument,
      operationName: "InvoiceAdjustmentReconciliationExceptions",
      variables: { input: queueInput(params) },
      signal: options.signal,
    });
    return safeParse(
      invoiceReconciliationPageSchema,
      response.invoiceAdjustmentReconciliationExceptions,
      "InvoiceReconciliationExceptionList",
    );
  }

  public async listBatches(params: InvoiceAdjustmentQueueParams, options: RequestOptions = {}) {
    const response = await requestGraphQL<{ invoiceAdjustmentBatches: unknown }>({
      document: InvoiceAdjustmentBatchesDocument,
      operationName: "InvoiceAdjustmentBatches",
      variables: { input: queueInput(params) },
      signal: options.signal,
    });
    return safeParse(
      invoiceAdjustmentBatchPageSchema,
      response.invoiceAdjustmentBatches,
      "InvoiceAdjustmentBatchList",
    );
  }

  public async getSummary(options: RequestOptions = {}) {
    const response = await requestGraphQL<{ invoiceAdjustmentOperationsSummary: unknown }>({
      document: InvoiceAdjustmentWorkloadSummaryDocument,
      operationName: "InvoiceAdjustmentWorkloadSummary",
      variables: {},
      signal: options.signal,
    });
    return safeParse(
      invoiceAdjustmentOperationsSummarySchema,
      response.invoiceAdjustmentOperationsSummary,
      "InvoiceAdjustmentOperationsSummary",
    );
  }
}
