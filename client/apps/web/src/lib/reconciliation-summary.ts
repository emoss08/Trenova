import type { ReconciliationSummary } from "@/types/bank-receipt";

/**
 * Whether anything has ever reached the reconciliation programme. A summary
 * of zeros is not a clean book; it is a book nobody has opened, so the page
 * draws what it will become rather than a row of noughts.
 */
export function reconciliationHasActivity(summary: ReconciliationSummary): boolean {
  return (
    summary.importedCount > 0 ||
    summary.matchedCount > 0 ||
    summary.exceptionCount > 0 ||
    summary.activeWorkItemCount > 0 ||
    summary.assignedWorkItemCount > 0 ||
    summary.inReviewWorkItemCount > 0
  );
}

/** Matched receipts as a whole percentage of those imported; zero until something is imported. */
export function matchRate(summary: ReconciliationSummary): number {
  return summary.importedCount > 0
    ? Math.round((summary.matchedCount / summary.importedCount) * 100)
    : 0;
}
