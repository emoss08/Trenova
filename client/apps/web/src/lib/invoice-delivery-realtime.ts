import {
  resolveEntityID,
  type ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";

export const INVOICE_RESOURCE = "invoice";
export const INVOICE_PDF_GENERATED_ACTION = "pdf.generated";
export const INVOICE_SEND_UPDATED_ACTION = "send.updated";

/** What the open invoice last showed, so a repeated event announces nothing new. */
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
  outcome: InvoiceDeliveryOutcome | null;
  seen: InvoiceDeliveryState;
}

function stringField(entity: Record<string, unknown>, field: string): string | null {
  const value = entity[field];
  return typeof value === "string" && value.trim() !== "" ? value : null;
}

/**
 * Whether an invoice event finishes something the person with that invoice
 * open is waiting on: its PDF becoming ready, or an email send settling as
 * sent, partly sent or failed. A send moves through several events (queued,
 * attempted, then the provider's answer), and the same failure can be
 * reported more than once, so only a change from what was last seen counts.
 */
export function invoiceDeliveryOutcome(
  event: ResourceInvalidationEvent,
  invoiceId: string,
  seen: InvoiceDeliveryState,
): InvoiceDeliveryDecision {
  const entity = event.entity;
  if (event.resource !== INVOICE_RESOURCE || !entity || resolveEntityID(event) !== invoiceId) {
    return { outcome: null, seen };
  }

  const next: InvoiceDeliveryState = {
    sendStatus: stringField(entity, "sendStatus") ?? seen.sendStatus,
    pdfDocumentId: stringField(entity, "pdfDocumentId") ?? seen.pdfDocumentId,
  };
  const number = stringField(entity, "number") ?? "";
  const error = stringField(entity, "lastSendError");

  if (event.action === INVOICE_PDF_GENERATED_ACTION) {
    const ready = next.pdfDocumentId !== null && next.pdfDocumentId !== seen.pdfDocumentId;
    return { outcome: ready ? { kind: "pdf-ready", number } : null, seen: next };
  }

  if (event.action !== INVOICE_SEND_UPDATED_ACTION || next.sendStatus === seen.sendStatus) {
    return { outcome: null, seen: next };
  }

  switch (next.sendStatus) {
    case "Sent":
      return { outcome: { kind: "sent", number }, seen: next };
    case "PartiallySent":
      return { outcome: { kind: "partially-sent", number, error }, seen: next };
    case "Failed":
      return { outcome: { kind: "failed", number, error }, seen: next };
    default:
      return { outcome: null, seen: next };
  }
}
