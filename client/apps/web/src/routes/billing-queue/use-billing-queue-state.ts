import { createParser, parseAsBoolean, parseAsString, parseAsStringLiteral } from "nuqs";

const parseAsStringArray = createParser<string[]>({
  parse: (value) => {
    if (!value) return [];
    try {
      return JSON.parse(value) as string[];
    } catch {
      return [];
    }
  },
  serialize: (value) => {
    if (!value || value.length === 0) return "";
    return JSON.stringify(value);
  },
}).withDefault([]);

/**
 * The two halves of the billing queue.
 *
 * "shipments" is the per-shipment approval machine. "statements" is the same
 * freight seen through each customer's billing schedule — what is accumulating
 * onto their next consolidated invoice. They share a route because a biller
 * moves between them constantly: approve here, watch it land there.
 */
export const BILLING_QUEUE_VIEWS = ["shipments", "statements"] as const;
export type BillingQueueView = (typeof BILLING_QUEUE_VIEWS)[number];

export const queueSearchParamsParser = {
  view: parseAsStringLiteral(BILLING_QUEUE_VIEWS).withDefault("shipments"),
  item: parseAsString,
  customer: parseAsString,
  status: parseAsString,
  query: parseAsString.withDefault(""),
  billType: parseAsString,
  billers: parseAsStringArray,
  includePosted: parseAsBoolean.withDefault(false),
  preset: parseAsString,
};

export const queueViewSearchParamsParser = {
  view: queueSearchParamsParser.view,
};

export const queueSelectionSearchParamsParser = {
  item: queueSearchParamsParser.item,
};

export const statementSelectionSearchParamsParser = {
  customer: queueSearchParamsParser.customer,
};

export const queueToolbarSearchParamsParser = {
  status: queueSearchParamsParser.status,
  includePosted: queueSearchParamsParser.includePosted,
};

export const queueSidebarSearchParamsParser = {
  status: queueSearchParamsParser.status,
  query: queueSearchParamsParser.query,
  billType: queueSearchParamsParser.billType,
  billers: queueSearchParamsParser.billers,
  includePosted: queueSearchParamsParser.includePosted,
  preset: queueSearchParamsParser.preset,
};
