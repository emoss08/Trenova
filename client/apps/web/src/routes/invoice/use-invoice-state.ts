import { parseAsString } from "nuqs";

export const invoiceSearchParamsParser = {
  item: parseAsString,
  status: parseAsString,
  query: parseAsString.withDefault(""),
  billType: parseAsString,
};

export const invoiceSelectionSearchParamsParser = {
  item: invoiceSearchParamsParser.item,
};

export const invoiceSidebarSearchParamsParser = {
  status: invoiceSearchParamsParser.status,
  query: invoiceSearchParamsParser.query,
  billType: invoiceSearchParamsParser.billType,
};
