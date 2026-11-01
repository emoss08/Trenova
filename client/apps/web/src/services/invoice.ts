import {
  GenerateInvoicePdfDocument,
  InvoiceDetailDocument,
  InvoiceEmailAttemptsDocument,
  InvoicePostPreviewDocument,
  InvoiceSendPlanDocument,
  PostInvoiceDocument,
  SendInvoiceDocument,
  UpdateInvoiceDraftDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  generateInvoicePdfResultSchema,
  invoiceEmailAttemptPageSchema,
  invoicePostPreviewSchema,
  invoiceSchema,
  invoiceSendPlanSchema,
  invoiceSendResultSchema,
  updateInvoiceDraftSchema,
  type UpdateInvoiceDraft,
} from "@trenova/shared/types/invoice";

type RequestOptions = { signal?: AbortSignal };

export class InvoiceService {
  public async getById(id: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ invoice: unknown }>({
      document: InvoiceDetailDocument,
      operationName: "InvoiceDetail",
      variables: { id },
      signal: options.signal,
    });
    return safeParse(invoiceSchema, data.invoice, "Invoice");
  }

  public async updateDraft(id: string, data: UpdateInvoiceDraft) {
    const input = updateInvoiceDraftSchema.parse(data);
    const response = await requestGraphQL<{ updateInvoiceDraft: unknown }>({
      document: UpdateInvoiceDraftDocument,
      operationName: "UpdateInvoiceDraft",
      variables: { invoiceId: id, input },
    });
    return safeParse(invoiceSchema, response.updateInvoiceDraft, "Invoice");
  }

  public async generatePdf(id: string) {
    const data = await requestGraphQL<{ generateInvoicePdf: unknown }>({
      document: GenerateInvoicePdfDocument,
      operationName: "GenerateInvoicePdf",
      variables: { invoiceId: id },
    });
    return safeParse(
      generateInvoicePdfResultSchema,
      data.generateInvoicePdf,
      "GenerateInvoicePdfResult",
    );
  }

  public async getSendPlan(id: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ invoiceSendPlan: unknown }>({
      document: InvoiceSendPlanDocument,
      operationName: "InvoiceSendPlan",
      variables: { invoiceId: id },
      signal: options.signal,
    });
    return safeParse(invoiceSendPlanSchema, data.invoiceSendPlan, "InvoiceSendPlan");
  }

  /** Emails the invoice; a later call sends it again after a failed or earlier send. */
  public async send(id: string) {
    const data = await requestGraphQL<{ sendInvoice: unknown }>({
      document: SendInvoiceDocument,
      operationName: "SendInvoice",
      variables: { invoiceId: id },
    });
    return safeParse(invoiceSendResultSchema, data.sendInvoice, "InvoiceSendResult");
  }

  public async listEmailAttempts(
    id: string,
    params: { limit?: number; offset?: number } = {},
    options: RequestOptions = {},
  ) {
    const data = await requestGraphQL<{ invoiceEmailAttempts: unknown }>({
      document: InvoiceEmailAttemptsDocument,
      operationName: "InvoiceEmailAttempts",
      variables: { invoiceId: id, limit: params.limit, offset: params.offset },
      signal: options.signal,
    });
    return safeParse(
      invoiceEmailAttemptPageSchema,
      data.invoiceEmailAttempts,
      "InvoiceEmailAttempts",
    );
  }

  /** What posting the invoice would book and move, without posting it. */
  public async getPostPreview(id: string, options: RequestOptions = {}) {
    const data = await requestGraphQL<{ invoicePostPreview: unknown }>({
      document: InvoicePostPreviewDocument,
      operationName: "InvoicePostPreview",
      variables: { invoiceId: id },
      signal: options.signal,
    });
    return safeParse(invoicePostPreviewSchema, data.invoicePostPreview, "InvoicePostPreview");
  }

  public async post(id: string) {
    const data = await requestGraphQL<{ postInvoice: unknown }>({
      document: PostInvoiceDocument,
      operationName: "PostInvoice",
      variables: { invoiceId: id },
    });
    return safeParse(invoiceSchema, data.postInvoice, "Invoice");
  }
}
