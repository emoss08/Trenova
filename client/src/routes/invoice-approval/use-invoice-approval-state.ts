import { parseAsString } from "nuqs";

export const invoiceApprovalSearchParamsParser = {
  item: parseAsString,
  query: parseAsString.withDefault(""),
  kind: parseAsString,
};
