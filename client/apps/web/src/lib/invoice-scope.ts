import type { InvoiceScope } from "@trenova/shared/types/invoice";
import { periodRange } from "./billing-schedule";

type InvoicePeriodFields = {
  scope: InvoiceScope;
  periodStart?: number | null;
  periodEnd?: number | null;
};

/**
 * Whether the invoice's `shipmentId` and `billingQueueItemId` name what it bills.
 *
 * An order or consolidated invoice bills several shipments; its billing-queue item
 * is only the anchor leg that backs the invoice's single-valued foreign key, so
 * linking to it as "the" shipment or queue item would send a biller to one leg of
 * many.
 */
export function invoiceBillsSingleShipment(scope: InvoiceScope): boolean {
  return scope === "Shipment" || scope === "Adjustment";
}

/**
 * The billing period a consolidated invoice covers, formatted for display, or
 * null for every invoice that does not state one.
 */
export function invoiceBillingPeriod(invoice: InvoicePeriodFields): string | null {
  if (invoice.scope !== "Consolidated" || !invoice.periodStart || !invoice.periodEnd) {
    return null;
  }

  return periodRange(invoice.periodStart, invoice.periodEnd);
}
