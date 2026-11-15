import {
  parseDateWindow,
  serializeDateWindow,
  type DateWindow,
} from "@/lib/billing-queue-date-window";
import { BILLING_QUEUE_OPEN_LANES } from "@trenova/shared/types/billing-queue";
import {
  searchBoolean,
  searchCodec,
  searchInteger,
  searchList,
  searchLiteral,
  searchString,
} from "@trenova/shared/lib/router/search-codec";

const stringArrayCodec = searchCodec<string[]>({
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

/** A billing queue date filter in the URL, `last:7` or `range:FROM:TO`. */
const dateWindowCodec = searchCodec<DateWindow>({
  parse: (value) => parseDateWindow(value),
  serialize: (value) => serializeDateWindow(value),
});

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
  view: searchLiteral(BILLING_QUEUE_VIEWS).withDefault("shipments"),
  item: searchString,
  customer: searchString,
  status: searchString,
  query: searchString.withDefault(""),
  billType: searchString,
  billers: stringArrayCodec,
  /** Narrows the queue to the items billed to one customer, on any shipment. */
  payer: searchString,
  includeClosed: searchBoolean.withDefault(false),
  preset: searchString,
  /** The lanes the list shows; empty shows every open lane. */
  lanes: searchList(searchLiteral(BILLING_QUEUE_OPEN_LANES)).withDefault([]),
  minAmount: searchString,
  maxAmount: searchString,
  /** Only items queued before this Unix time; set by "older than N days". */
  createdBefore: searchInteger,
  /** Narrows the queue to one customer's shipments, whoever each bill goes to. */
  shipmentCustomer: searchString,
  /** Only shipments delivered, picked up, or items queued within these windows. */
  delivered: dateWindowCodec,
  pickedUp: dateWindowCodec,
  queued: dateWindowCodec,
  /**
   * The transfer run the bulk dialog is watching, while the dialog is open. It
   * lives in the URL so the toast announcing a finished transfer can link straight
   * back to its report, and so a refresh mid-run reattaches instead of losing sight
   * of it; closing the dialog drops it.
   */
  transferRun: searchString,
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

export const transferRunSearchParamsParser = {
  transferRun: queueSearchParamsParser.transferRun,
};

export const queueSidebarSearchParamsParser = {
  status: queueSearchParamsParser.status,
  query: queueSearchParamsParser.query,
  billType: queueSearchParamsParser.billType,
  billers: queueSearchParamsParser.billers,
  payer: queueSearchParamsParser.payer,
  includeClosed: queueSearchParamsParser.includeClosed,
  preset: queueSearchParamsParser.preset,
  lanes: queueSearchParamsParser.lanes,
  minAmount: queueSearchParamsParser.minAmount,
  maxAmount: queueSearchParamsParser.maxAmount,
  createdBefore: queueSearchParamsParser.createdBefore,
  shipmentCustomer: queueSearchParamsParser.shipmentCustomer,
  delivered: queueSearchParamsParser.delivered,
  pickedUp: queueSearchParamsParser.pickedUp,
  queued: queueSearchParamsParser.queued,
};
