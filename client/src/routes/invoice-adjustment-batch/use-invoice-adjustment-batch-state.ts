import { parseAsString } from "nuqs";

export const invoiceAdjustmentBatchSearchParamsParser = {
  item: parseAsString,
  query: parseAsString.withDefault(""),
  status: parseAsString,
};
