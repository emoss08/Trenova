import { parseAsString } from "nuqs";

export const invoiceSearchParamsParser = {
  item: parseAsString,
  status: parseAsString,
  query: parseAsString.withDefault(""),
  billType: parseAsString,
};
