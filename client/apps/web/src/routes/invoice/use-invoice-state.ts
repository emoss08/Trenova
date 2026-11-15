import {
  INVOICE_DETAIL_TABS,
  type InvoiceDetailTab,
} from "@trenova/shared/types/invoice-detail-tab";
import {
  searchBoolean,
  searchLiteral,
  searchString,
} from "@trenova/shared/lib/router/search-codec";

export function isInvoiceDetailTab(value: unknown): value is InvoiceDetailTab {
  return typeof value === "string" && (INVOICE_DETAIL_TABS as readonly string[]).includes(value);
}

export const invoiceSearchParamsParser = {
  item: searchString,
  tab: searchLiteral(INVOICE_DETAIL_TABS).withDefault("overview"),
  status: searchString,
  query: searchString.withDefault(""),
  billType: searchString,
  scope: searchString,
  dispute: searchBoolean.withDefault(false),
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
