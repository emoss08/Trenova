import type { BillingTransferRun, BillingTransferRunStatus } from "@/lib/graphql/billing-transfer";

/**
 * Reading a transfer run.
 *
 * The run lives on the server now, so the dialog is a reader rather than the
 * thing doing the work. What lives here is the translation of a run row into
 * the questions the dialog asks of it — nothing here recomputes what the server
 * already counted.
 */

const TERMINAL_STATUSES: ReadonlySet<BillingTransferRunStatus> = new Set([
  "Completed",
  "Canceled",
  "Failed",
]);

/** Whether the run has finished, whatever the outcome. */
export function isRunTerminal(run: BillingTransferRun | null | undefined): boolean {
  return run ? TERMINAL_STATUSES.has(run.status) : false;
}

/**
 * Whether the dialog should keep asking.
 *
 * A finished run never changes again, and polling one forever is how a
 * background job quietly becomes a load on the database.
 */
export function shouldPollRun(run: BillingTransferRun | null | undefined): boolean {
  if (!run) return false;

  return !isRunTerminal(run);
}

/**
 * How far through the run is, as a fraction.
 *
 * A run that has not been told how much there is to do reports nothing rather
 * than zero. An empty bar reads as "no progress", which is the wrong statement
 * about a run that is still working out which shipments it has to touch.
 */
export function runProgress(run: BillingTransferRun | null | undefined): number | null {
  if (!run || run.totalCount <= 0) return null;

  return Math.min(1, run.processedCount / run.totalCount);
}

/**
 * Whether a stop has been asked for but has not taken effect yet.
 *
 * The batch in flight always finishes, so there is a stretch where the run is
 * still moving shipments after the biller pressed Stop. Saying so is better
 * than a button that looks broken.
 */
export function isRunStopping(run: BillingTransferRun | null | undefined): boolean {
  if (!run) return false;

  return run.cancelRequestedAt !== null && !isRunTerminal(run);
}

/** Whether a retry would attempt anything. */
export function canRetryRun(run: BillingTransferRun | null | undefined): boolean {
  if (!run || !isRunTerminal(run)) return false;

  return run.retryableCount > 0 || run.skippedCount > 0;
}

export type BillingTransferRunSummary = {
  transferred: number;
  markedReadyToInvoice: number;
  notTransferred: number;
  notProcessed: number;
  total: number;
  retryable: number;
  unmatched: number;
};

export function summarizeRun(run: BillingTransferRun): BillingTransferRunSummary {
  return {
    transferred: run.transferredCount,
    markedReadyToInvoice: run.markedReadyToInvoiceCount,
    notTransferred: run.notTransferredCount,
    notProcessed: run.skippedCount,
    total: run.totalCount,
    retryable: run.retryableCount,
    unmatched: run.unmatchedCount,
  };
}

/**
 * The most shipments one run carries. A search matching more than this reports
 * the remainder as unmatched rather than silently transferring a prefix.
 */
export const MAX_BILLING_TRANSFER_CANDIDATE_IDS = 5000;
