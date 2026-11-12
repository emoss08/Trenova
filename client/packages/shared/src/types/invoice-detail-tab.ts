import { z } from "zod";

/** The tabs of an invoice's detail pane, which a link to an invoice can open on. */
export const INVOICE_DETAIL_TABS = [
  "overview",
  "delivery",
  "charges",
  "payments",
  "disputes",
  "documents",
  "activity",
] as const;
export type InvoiceDetailTab = (typeof INVOICE_DETAIL_TABS)[number];

export const invoiceDetailTabSchema = z.enum(INVOICE_DETAIL_TABS);
