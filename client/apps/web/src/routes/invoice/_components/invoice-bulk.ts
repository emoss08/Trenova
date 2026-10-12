import type { InvoiceTableRowFieldsFragment } from "@trenova/graphql/generated/graphql";

type InvoiceRow = Pick<InvoiceTableRowFieldsFragment, "id" | "status" | "sendStatus">;

export type InvoiceBulkEligibility<T extends InvoiceRow> = {
  postable: T[];
  /** Posted invoices that have not gone out, or whose last delivery failed. */
  sendable: T[];
};

const UNSENT = new Set<InvoiceRow["sendStatus"]>(["NotSent", "Failed", "PartiallySent"]);

/**
 * Splits a selection by what can be done to each invoice. A draft is never
 * sent, and a bulk send never repeats an invoice that already went out or is
 * on its way, so one click cannot email a customer the same invoice twice.
 */
export function invoiceBulkEligibility<T extends InvoiceRow>(
  invoices: readonly T[],
): InvoiceBulkEligibility<T> {
  const result: InvoiceBulkEligibility<T> = { postable: [], sendable: [] };

  for (const invoice of invoices) {
    if (invoice.status === "Draft") {
      result.postable.push(invoice);
    } else if (invoice.status === "Posted" && UNSENT.has(invoice.sendStatus)) {
      result.sendable.push(invoice);
    }
  }

  return result;
}
