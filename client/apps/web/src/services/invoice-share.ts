import {
  InvoiceShareCandidatesDocument,
  InvoiceSharesDocument,
  ShareInvoiceDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { z } from "zod";
import {
  invoiceShareSchema,
  invoiceShareUserSchema,
  shareInvoiceResultSchema,
  type ShareInvoicePayload,
} from "@trenova/shared/types/invoice-share";

const CANDIDATE_LIMIT = 10;

export class InvoiceShareService {
  public async list(invoiceId: string, options: { signal?: AbortSignal } = {}) {
    const data = await requestGraphQL<{ invoiceShares: unknown }>({
      document: InvoiceSharesDocument,
      operationName: "InvoiceShares",
      variables: { invoiceId },
      signal: options.signal,
    });
    return safeParse(z.array(invoiceShareSchema), data.invoiceShares ?? [], "InvoiceShareList");
  }

  public async candidates(invoiceId: string, query: string, signal?: AbortSignal) {
    const trimmed = query.trim();
    const data = await requestGraphQL<{ invoiceShareCandidates: unknown }>({
      document: InvoiceShareCandidatesDocument,
      operationName: "InvoiceShareCandidates",
      variables: { invoiceId, query: trimmed || undefined, limit: CANDIDATE_LIMIT },
      signal,
    });
    return safeParse(
      z.array(invoiceShareUserSchema),
      data.invoiceShareCandidates ?? [],
      "InvoiceShareCandidateList",
    );
  }

  public async share(invoiceId: string, payload: ShareInvoicePayload) {
    const data = await requestGraphQL<{ shareInvoice: unknown }>({
      document: ShareInvoiceDocument,
      operationName: "ShareInvoice",
      variables: {
        invoiceId,
        input: { userIds: payload.userIds, note: payload.note.trim(), tab: payload.tab },
      },
    });
    return safeParse(shareInvoiceResultSchema, data.shareInvoice, "ShareInvoiceResult");
  }
}
