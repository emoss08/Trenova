import { translate } from "@trenova/shared/i18n/runtime";
import { toast } from "sonner";

export type BulkOutcomeFailure = {
  id: string;
  error: string;
};

export type BulkOutcome = {
  succeeded: string[];
  failed: BulkOutcomeFailure[];
};

export type BulkOutcomeMessages = {
  succeeded: (count: number) => string;
  partial: (succeeded: number, failed: number) => string;
  allFailed: (failed: number) => string;
  skipped?: number;
};

const MAX_ERRORS_IN_DESCRIPTION = 3;

function withSkipped(message: string, skipped: number): string {
  if (skipped <= 0) {
    return message;
  }
  return translate(
    "{0} ({1, plural, one {# ineligible skipped} other {# ineligible skipped}})",
    message,
    skipped,
  );
}

export function notifyBulkOutcome(
  result: BulkOutcome,
  { succeeded, partial, allFailed, skipped = 0 }: BulkOutcomeMessages,
) {
  if (result.failed.length === 0) {
    toast.success(withSkipped(succeeded(result.succeeded.length), skipped));
    return;
  }

  const description = result.failed
    .slice(0, MAX_ERRORS_IN_DESCRIPTION)
    .map((failure) => failure.error)
    .join("; ");
  if (result.succeeded.length === 0) {
    toast.error(withSkipped(allFailed(result.failed.length), skipped), { description });
    return;
  }
  toast.warning(withSkipped(partial(result.succeeded.length, result.failed.length), skipped), {
    description,
  });
}

/**
 * Runs one mutation per row, never short-circuiting, and folds the settled
 * results into a BulkOutcome for notifyBulkOutcome.
 */
export async function settleAll<T extends { id: string }>(
  rows: readonly T[],
  run: (row: T) => Promise<unknown>,
): Promise<BulkOutcome> {
  const results = await Promise.allSettled(rows.map((row) => run(row)));
  const outcome: BulkOutcome = { succeeded: [], failed: [] };
  results.forEach((result, index) => {
    const row = rows[index];
    if (result.status === "fulfilled") {
      outcome.succeeded.push(row.id);
    } else {
      outcome.failed.push({
        id: row.id,
        error: result.reason instanceof Error ? result.reason.message : String(result.reason),
      });
    }
  });
  return outcome;
}
