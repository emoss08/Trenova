import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { ShipmentBillingReadiness } from "@trenova/shared/types/shipment";

/** Why an item in review cannot be approved yet, most specific first. */
export type ApproveBlocker =
  | { kind: "detention"; count: number }
  | { kind: "rating" }
  | { kind: "review"; count: number }
  | { kind: "documents"; count: number }
  | { kind: "requirement"; message: string | null };

/**
 * What stands between an item in review and Approve, or null when nothing does. The
 * server refuses the approval for the same reasons; this is what the biller is told.
 */
export function approveBlocker(
  item: BillingQueueItem,
  readiness: ShipmentBillingReadiness | null | undefined,
): ApproveBlocker | null {
  if (item.detentionHolds.length > 0) {
    return { kind: "detention", count: item.detentionHolds.length };
  }
  if (item.ratingReviewHold) {
    return { kind: "rating" };
  }
  if (item.review?.blocker === "issue") {
    return { kind: "review", count: Math.max(item.review.needsCount, 1) };
  }
  if (readiness?.canMarkReadyToInvoice === false) {
    const missing = readiness.missingRequirements.length;
    if (missing > 0) {
      return { kind: "documents", count: missing };
    }
    return { kind: "requirement", message: readiness.validationFailures[0]?.message ?? null };
  }

  return null;
}
