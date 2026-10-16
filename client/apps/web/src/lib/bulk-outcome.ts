import { translate } from "@trenova/shared/i18n/runtime";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { toast } from "@trenova/shared/components/ui/toast";

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

export type SettleOptions<T> = {
  /** Requests kept in flight at once; every row starts at once when unset. */
  concurrency?: number;
  /** Names a failed row in the toast, so it reads "INV-12: period closed". */
  label?: (row: T) => string;
};

/** How many requests a billing bulk action keeps in flight at once. */
export const BULK_CONCURRENCY = 4;

function failureMessage(error: unknown): string {
  const description = describeError(error);
  return description.detail ?? description.message;
}

/**
 * Runs one mutation per row, never short-circuiting, and folds the settled
 * results into a BulkOutcome for notifyBulkOutcome: of fifty invoices, the
 * forty-eight that could be posted are, and the two that could not are named.
 */
export async function settleAll<T extends { id: string }>(
  rows: readonly T[],
  run: (row: T) => Promise<unknown>,
  { concurrency, label }: SettleOptions<T> = {},
): Promise<BulkOutcome> {
  const settled: PromiseSettledResult<unknown>[] = Array.from({ length: rows.length });
  let next = 0;
  const worker = async () => {
    while (next < rows.length) {
      const index = next;
      next += 1;
      try {
        settled[index] = { status: "fulfilled", value: await run(rows[index]) };
      } catch (reason) {
        settled[index] = { status: "rejected", reason };
      }
    }
  };
  const workers = Math.max(1, Math.min(concurrency ?? rows.length, rows.length));
  await Promise.all(Array.from({ length: workers }, worker));

  const outcome: BulkOutcome = { succeeded: [], failed: [] };
  settled.forEach((result, index) => {
    const row = rows[index];
    if (result.status === "fulfilled") {
      outcome.succeeded.push(row.id);
      return;
    }
    const message = failureMessage(result.reason);
    outcome.failed.push({ id: row.id, error: label ? `${label(row)}: ${message}` : message });
  });
  return outcome;
}
