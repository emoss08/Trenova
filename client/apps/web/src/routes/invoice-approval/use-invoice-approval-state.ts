import type { InvoiceAdjustmentKind } from "@trenova/graphql/generated/graphql";
import { parseAsString, parseAsStringLiteral } from "nuqs";

export const invoiceAdjustmentKinds = [
  "CreditOnly",
  "CreditAndRebill",
  "FullReversal",
  "WriteOff",
] as const satisfies readonly InvoiceAdjustmentKind[];

export const invoiceApprovalSearchParamsParser = {
  item: parseAsString,
  query: parseAsString.withDefault(""),
  kind: parseAsStringLiteral(invoiceAdjustmentKinds),
};
