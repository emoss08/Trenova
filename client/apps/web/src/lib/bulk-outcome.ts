import { toast } from "sonner";

export type BulkOutcomeFailure = {
  id: string;
  error: string;
};

export type BulkOutcome = {
  succeeded: string[];
  failed: BulkOutcomeFailure[];
};

export type BulkOutcomeLabels = {
  entity: string;
  verbPast: string;
  skipped?: number;
};

const MAX_ERRORS_IN_DESCRIPTION = 3;

function pluralize(count: number, entity: string): string {
  return `${count} ${entity}${count === 1 ? "" : "s"}`;
}

export function notifyBulkOutcome(
  result: BulkOutcome,
  { entity, verbPast, skipped = 0 }: BulkOutcomeLabels,
) {
  const skippedSuffix = skipped > 0 ? ` (${skipped} ineligible skipped)` : "";
  if (result.failed.length === 0) {
    toast.success(`${verbPast} ${pluralize(result.succeeded.length, entity)}${skippedSuffix}`);
    return;
  }

  const description = result.failed
    .slice(0, MAX_ERRORS_IN_DESCRIPTION)
    .map((failure) => failure.error)
    .join("; ");
  if (result.succeeded.length === 0) {
    toast.error(
      `All ${pluralize(result.failed.length, `selected ${entity}`)} failed${skippedSuffix}`,
      { description },
    );
    return;
  }
  toast.warning(
    `${verbPast} ${pluralize(result.succeeded.length, entity)}; ${result.failed.length} failed${skippedSuffix}`,
    { description },
  );
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
