import { recordPath } from "@/config/record-links";
import type { InvoiceDetailTab } from "@trenova/shared/types/invoice-detail-tab";

export function invoicePanelPath(invoiceId: string, tab: InvoiceDetailTab = "overview"): string {
  return recordPath("invoice", invoiceId, tab === "overview" ? undefined : { tab });
}
