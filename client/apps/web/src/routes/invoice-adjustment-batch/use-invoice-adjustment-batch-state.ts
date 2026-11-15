import { searchString } from "@trenova/shared/lib/router/search-codec";

export const invoiceAdjustmentBatchSearchParamsParser = {
  item: searchString,
  query: searchString.withDefault(""),
  status: searchString,
};
