import type { BillingQueueStatus } from "@trenova/shared/types/billing-queue";

type Statused = { status: BillingQueueStatus };

/** The selected items each bulk action can act on. */
export type BulkEligibility<T extends Statused> = {
  /** Not yet decided: can be approved, assigned or canceled. */
  open: T[];
  approvable: T[];
  holdable: T[];
  /** Approved and waiting for their invoice to be posted. */
  postable: T[];
};

const DECIDED = new Set<BillingQueueStatus>(["Approved", "Posted", "Canceled"]);

/**
 * Splits a selection by what can be done to each item, so a bulk action only
 * ever sends the items it applies to: posting a selection posts the approved
 * ones and leaves the rest, rather than failing on every item still in review.
 * The server checks each item again; this only keeps requests that are sure
 * to be refused from being sent.
 */
export function bulkEligibility<T extends Statused>(items: readonly T[]): BulkEligibility<T> {
  const result: BulkEligibility<T> = { open: [], approvable: [], holdable: [], postable: [] };

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
