import { searchString } from "@trenova/shared/lib/router/search-codec";

export const invoiceReconciliationSearchParamsParser = {
  item: searchString,
  query: searchString.withDefault(""),
  status: searchString,
};
