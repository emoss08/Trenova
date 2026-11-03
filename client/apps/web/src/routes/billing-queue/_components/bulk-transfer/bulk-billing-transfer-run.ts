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

/**
 * What a finished run amounts to, which decides the tone and the headline of its report.
 *
 * - `empty`: nothing matched, so there was nothing to move.
 * - `failed` and `canceled`: the run ended before it was done, whatever it moved first.
 * - `transferred`: every shipment moved.
 * - `partial`: some moved and some were held back or never reached.
 * - `blocked`: none moved.
 */
export type RunOutcome = "empty" | "failed" | "canceled" | "transferred" | "partial" | "blocked";

export function runOutcome(run: BillingTransferRun): RunOutcome {
  if (run.status === "Failed") return "failed";
  if (run.status === "Canceled") return "canceled";
  if (run.totalCount === 0) return "empty";
  if (run.transferredCount === 0) return "blocked";
  if (run.transferredCount < run.totalCount) return "partial";
  return "transferred";
}

/**
 * How long the run has been working, or worked, in seconds. A run that has not started
 * has spent no time; a finished one stops the clock at its completion.
 */
export function runElapsedSeconds(run: BillingTransferRun, nowSeconds: number): number | null {
  if (!run.startedAt) return null;

  const end = run.completedAt ?? nowSeconds;
  return Math.max(0, end - run.startedAt);
}

/**
 * Roughly how long the rest of the run will take, from the pace so far. Nothing is promised
 * until the run has a pace worth projecting: a few shipments in, or a few seconds, the rate
 * swings too far to be a useful answer. The answer is rounded up to the minute, or to ten
 * seconds under a minute, so it holds still between polls instead of ticking with the clock.
 */
export function runSecondsLeft(run: BillingTransferRun, nowSeconds: number): number | null {
  if (isRunTerminal(run) || run.totalCount <= 0) return null;

  const elapsed = runElapsedSeconds(run, nowSeconds);
  if (elapsed === null || elapsed < MIN_PACE_SECONDS || run.processedCount < MIN_PACE_SHIPMENTS) {
    return null;
  }

  const remaining = Math.max(0, run.totalCount - run.processedCount);
  const seconds = (remaining * elapsed) / run.processedCount;
  return seconds < 60 ? Math.ceil(seconds / 10) * 10 : Math.ceil(seconds / 60) * 60;
}

const MIN_PACE_SECONDS = 5;
const MIN_PACE_SHIPMENTS = 10;

/**
 * The most shipments one run carries. A search matching more than this reports
 * the remainder as unmatched rather than silently transferring a prefix.
 */
export const MAX_BILLING_TRANSFER_CANDIDATE_IDS = 5000;
