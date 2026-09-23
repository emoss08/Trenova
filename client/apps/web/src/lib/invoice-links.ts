import { recordPath } from "@/config/record-links";
import type { InvoiceDetailTab } from "@trenova/shared/types/invoice-share";

export function invoicePanelPath(invoiceId: string, tab: InvoiceDetailTab = "overview"): string {
  return recordPath("invoice", invoiceId, tab === "overview" ? undefined : { tab });
}

export function absoluteAppUrl(path: string): string {
  return new URL(path, window.location.origin).toString();
}
