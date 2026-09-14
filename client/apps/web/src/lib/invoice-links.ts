import type { InvoiceDetailTab } from "@trenova/shared/types/invoice-share";

const INVOICES_PATH = "/billing/invoices";

export function invoicePanelPath(invoiceId: string, tab: InvoiceDetailTab = "overview"): string {
  const params = new URLSearchParams({ item: invoiceId });
  if (tab !== "overview") {
    params.set("tab", tab);
  }
  return `${INVOICES_PATH}?${params.toString()}`;
}

export function absoluteAppUrl(path: string): string {
  return new URL(path, window.location.origin).toString();
}
