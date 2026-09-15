import { INVOICE_DETAIL_TABS, type InvoiceDetailTab } from "@trenova/shared/types/invoice-share";
import { parseAsBoolean, parseAsString, parseAsStringLiteral } from "nuqs";

export function isInvoiceDetailTab(value: unknown): value is InvoiceDetailTab {
  return typeof value === "string" && (INVOICE_DETAIL_TABS as readonly string[]).includes(value);
}

export const invoiceSearchParamsParser = {
  item: parseAsString,
  tab: parseAsStringLiteral(INVOICE_DETAIL_TABS).withDefault("overview"),
  status: parseAsString,
  query: parseAsString.withDefault(""),
  billType: parseAsString,
  scope: parseAsString,
  dispute: parseAsBoolean.withDefault(false),
};

export const invoiceSelectionSearchParamsParser = {
  item: invoiceSearchParamsParser.item,
};

export const invoiceDetailTabSearchParamsParser = {
  tab: invoiceSearchParamsParser.tab,
};

export const invoiceSidebarSearchParamsParser = {
  status: invoiceSearchParamsParser.status,
  query: invoiceSearchParamsParser.query,
  billType: invoiceSearchParamsParser.billType,
  scope: invoiceSearchParamsParser.scope,
  dispute: invoiceSearchParamsParser.dispute,
};
