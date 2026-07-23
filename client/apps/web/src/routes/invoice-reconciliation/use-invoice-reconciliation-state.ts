import { parseAsString } from "nuqs";

export const invoiceReconciliationSearchParamsParser = {
  item: parseAsString,
  query: parseAsString.withDefault(""),
  status: parseAsString,
};
