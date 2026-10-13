import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";

/** The selected items each bulk action can act on. */
export type BulkEligibility = {
  /** Not yet decided: can be approved, assigned or canceled. */
  open: BillingQueueItem[];
  approvable: BillingQueueItem[];
  holdable: BillingQueueItem[];
  /** Approved and waiting for their invoice to be posted. */
  postable: BillingQueueItem[];
};

const DECIDED = new Set<BillingQueueItem["status"]>(["Approved", "Posted", "Canceled"]);

/**
 * Splits a selection by what can be done to each item, so a bulk action only
 * ever sends the items it applies to: posting a selection posts the approved
 * ones and leaves the rest, rather than failing on every item still in review.
 * The server checks each item again; this only keeps requests that are sure
 * to be refused from being sent.
 */
export function bulkEligibility(items: readonly BillingQueueItem[]): BulkEligibility {
  const result: BulkEligibility = { open: [], approvable: [], holdable: [], postable: [] };

  for (const item of items) {
    if (item.status === "Approved") {
      result.postable.push(item);
    }
    if (DECIDED.has(item.status)) {
      continue;
    }
    result.open.push(item);
    if (item.status !== "OnHold") {
      result.approvable.push(item);
      result.holdable.push(item);
    }
  }

  return result;
}
