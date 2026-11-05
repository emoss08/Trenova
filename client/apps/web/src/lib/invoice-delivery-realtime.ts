import {
  resolveEntityID,
  type ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import type { Invoice } from "@trenova/shared/types/invoice";

export const INVOICE_RESOURCE = "invoice";
export const INVOICE_PDF_GENERATED_ACTION = "pdf.generated";
export const INVOICE_SEND_UPDATED_ACTION = "send.updated";
export const INVOICE_POST_FAILED_ACTION = "post.failed";

export type DeliveryInvoice = Pick<
  Invoice,
  "id" | "number" | "sendStatus" | "pdfDocumentId" | "lastSendError"
>;

/** What the open invoice last showed, so a refetch that moved nothing announces nothing. */
export interface InvoiceDeliveryState {
  sendStatus: string;
  pdfDocumentId: string | null;
}

export type InvoiceDeliveryOutcome =
  | { kind: "pdf-ready"; number: string }
  | { kind: "sent"; number: string }
  | { kind: "partially-sent"; number: string; error: string | null }
  | { kind: "failed"; number: string; error: string | null };

export interface InvoiceDeliveryDecision {
  outcomes: InvoiceDeliveryOutcome[];
  seen: InvoiceDeliveryState;
}

/**
 * What the open invoice finished since it last showed: its PDF becoming ready
 * or an email send settling as sent, partly sent or failed. It reads the
 * invoice the page refetched, because an event names the invoice and never
 * what it now says.
 */
export function invoiceDeliveryOutcomes(
  invoice: DeliveryInvoice,
  seen: InvoiceDeliveryState,
): InvoiceDeliveryDecision {
  const next: InvoiceDeliveryState = {
    sendStatus: invoice.sendStatus,
    pdfDocumentId: invoice.pdfDocumentId ?? null,
  };
  const number = invoice.number;
  const error = invoice.lastSendError ?? null;
  const outcomes: InvoiceDeliveryOutcome[] = [];

  if (next.pdfDocumentId !== null && next.pdfDocumentId !== seen.pdfDocumentId) {
    outcomes.push({ kind: "pdf-ready", number });
  }

  if (next.sendStatus !== seen.sendStatus) {
    switch (next.sendStatus) {
      case "Sent":
        outcomes.push({ kind: "sent", number });
        break;
      case "PartiallySent":
        outcomes.push({ kind: "partially-sent", number, error });
        break;
      case "Failed":
        outcomes.push({ kind: "failed", number, error });
        break;
    }
  }

  return { outcomes, seen: next };
}

/**
 * Whether an event says background work on this invoice just moved. The page
 * announces only what such an event brought, so a stale cached invoice catching
 * up on open reads as old news.
 */
export function isInvoiceDeliveryEvent(
  event: ResourceInvalidationEvent,
  invoiceId: string,
): boolean {
  return (
    event.resource === INVOICE_RESOURCE &&
    (event.action === INVOICE_PDF_GENERATED_ACTION ||
      event.action === INVOICE_SEND_UPDATED_ACTION) &&
    resolveEntityID(event) === invoiceId
  );
}

/**
 * Whether an event says an automatic post of this invoice gave up, and who has
 * already been told, so that person is not told twice.
 */
export function invoicePostFailure(
  event: ResourceInvalidationEvent,
  invoiceId: string,
): { notifiedUserId: string | null } | null {
  if (
    event.resource !== INVOICE_RESOURCE ||
    event.action !== INVOICE_POST_FAILED_ACTION ||
    resolveEntityID(event) !== invoiceId
  ) {
    return null;
  }

  const notified = event.entity?.notifiedUserId;

  return { notifiedUserId: typeof notified === "string" && notified !== "" ? notified : null };
}
