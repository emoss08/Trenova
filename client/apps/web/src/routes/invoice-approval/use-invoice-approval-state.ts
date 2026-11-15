import type { InvoiceAdjustmentKind } from "@trenova/graphql/generated/graphql";
import { searchLiteral, searchString } from "@trenova/shared/lib/router/search-codec";

export const invoiceAdjustmentKinds = [
  "CreditOnly",
  "CreditAndRebill",
  "FullReversal",
  "WriteOff",
] as const satisfies readonly InvoiceAdjustmentKind[];

export const invoiceApprovalSearchParamsParser = {
  item: searchString,
  query: searchString.withDefault(""),
  kind: searchLiteral(invoiceAdjustmentKinds),
};
