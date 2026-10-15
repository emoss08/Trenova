import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** The parts of an invoice's EDI send plan that decide whether a 210 can go now. */
export type InvoiceEdiPlanGate = {
  enabled: boolean;
  blockers: readonly string[];
  status: string;
};

/** The parts of an invoice that decide whether it can be emailed now. */
export type InvoiceEmailGate = {
  status: string;
  sendStatus?: string | null;
  pdfDocumentId?: string | null;
};

/**
 * Why an EDI 210 cannot go for the invoice right now, in the customer's terms,
 * or null when it can. The plan's own blockers come first; a missing plan
 * means it has not loaded yet.
 */
export function invoiceEdiSendBlocker(
  invoiceStatus: string,
  plan: InvoiceEdiPlanGate | null | undefined,
  t: TranslateFn,
): string | null {
  if (!plan) return t("Checking the customer's EDI configuration.");
  if (!plan.enabled) {
    return plan.blockers[0] ?? t("EDI invoicing is not enabled for this customer.");
  }
  if (plan.blockers.length > 0) return plan.blockers[0];
  if (invoiceStatus !== "Posted") return t("Only a posted invoice can be sent by EDI.");
  if (plan.status === "Queued" || plan.status === "Sending") {
    return t("An EDI send for this invoice is already in progress.");
  }
  return null;
}

/**
 * Why the invoice cannot be emailed right now, or null when it can: never a
 * draft or a voided one, never without its PDF, never twice at once, and
 * never while the send plan has an error.
 */
export function invoiceEmailSendBlocker(
  invoice: InvoiceEmailGate,
  plan: { errors: readonly string[] } | null | undefined,
  planLoading: boolean,
  t: TranslateFn,
): string | null {
  if (invoice.status === "Voided") {
    return t("A voided invoice is never sent.");
  }
  if (invoice.status === "Draft") {
    return t("Post the invoice before sending it. Draft invoices are never emailed.");
  }
  if (planLoading) {
    return t("Checking invoice email configuration.");
  }
  if (!invoice.pdfDocumentId) {
    return t("Generate the invoice PDF before sending.");
  }
  if (invoice.sendStatus === "Sending") {
    return t("This invoice is already being sent. Wait for that send to finish.");
  }
  if (!plan) {
    return t("Send plan is unavailable.");
  }
  if (plan.errors.length > 0) {
    return plan.errors.join("; ");
  }
  return null;
}
