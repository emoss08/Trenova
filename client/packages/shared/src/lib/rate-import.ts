import type {
  RateImportBatch,
  RateImportChange,
  RateImportChangeKind,
  RateImportSummary,
} from "../types/rate";

/**
 * Reading a staged rate import.
 *
 * The dry run is the point of the feature, and it is only useful if somebody
 * reads it. What lives here is the translation of a diff into the order and the
 * words a person checks in — nothing here recomputes what the server decided.
 */

/** Whether this import is still waiting to be applied. */
export function canCommit(batch: RateImportBatch | undefined): boolean {
  return batch?.status === "Parsed";
}

const KIND_LABEL: Record<RateImportChangeKind, string> = {
  Removed: "Stops pricing",
  Added: "New lane",
  Changed: "Rate changes",
  Duplicate: "Listed twice",
  Unchanged: "No change",
};

export function changeKindLabel(kind: RateImportChangeKind): string {
  return KIND_LABEL[kind] ?? kind;
}

/**
 * How a change should read.
 *
 * A lane that stops pricing is the one that has to stand out: it is the only
 * change where doing nothing produces a shipment nobody can invoice.
 */
export type ChangeTone = "danger" | "warning" | "neutral" | "quiet";

const KIND_TONE: Record<RateImportChangeKind, ChangeTone> = {
  Removed: "danger",
  Duplicate: "warning",
  Added: "neutral",
  Changed: "neutral",
  Unchanged: "quiet",
};

export function changeTone(kind: RateImportChangeKind): ChangeTone {
  return KIND_TONE[kind] ?? "neutral";
}

/**
 * The changes worth reading, which is everything except the lanes a sheet
 * restated unchanged.
 *
 * A monthly sheet is mostly unchanged lanes, and showing all of them buries the
 * handful that moved.
 */
export function meaningfulChanges(
  changes: readonly RateImportChange[] | null | undefined,
): RateImportChange[] {
  return (changes ?? []).filter((change) => change.kind !== "Unchanged");
}

/**
 * What one changed lane's movement reads as.
 *
 * Every term that moved is listed, because somebody approving a sheet needs all
 * of them rather than whichever was compared first. A term the sheet drops
 * says so rather than showing an empty value.
 */
export function describeFieldChange(change: {
  field: string;
  before: string;
  after: string;
}): string {
  if (!change.before) return `${change.field} set to ${change.after}`;
  if (!change.after) return `${change.field} removed (was ${change.before})`;

  return `${change.field} ${change.before} → ${change.after}`;
}

/**
 * A one-line reading of what committing this sheet would do.
 *
 * A sheet that changes nothing says so: it usually means somebody uploaded last
 * month's file again, and a screen full of "no change" rows does not make that
 * obvious.
 */
export function importHeadline(batch: RateImportBatch | undefined): string {
  if (!batch) return "";

  if (batch.status === "Failed") {
    return batch.error || "This file could not be read.";
  }

  if (batch.status === "Committed") {
    return "This sheet has been applied to the agreement.";
  }

  if (batch.status === "Discarded") {
    return "This sheet was reviewed and not applied.";
  }

  const summary = batch.summary;
  if (!summary || !changesAnything(summary)) {
    return "This sheet would not change anything. It may be a file that has already been imported.";
  }

  const parts: string[] = [];
  if (summary.added > 0) parts.push(`${summary.added} new`);
  if (summary.changed > 0) parts.push(`${summary.changed} changed`);
  if (summary.removed > 0) parts.push(`${summary.removed} would stop pricing`);

  return `Committing this would leave the agreement with ${parts.join(", ")}.`;
}

export function changesAnything(summary: RateImportSummary | null | undefined): boolean {
  if (!summary) return false;

  return summary.added > 0 || summary.changed > 0 || summary.removed > 0;
}

/**
 * What to warn about before somebody commits.
 *
 * These are the things a person would want to have been told rather than
 * discover afterwards, so they are stated up front rather than left to be
 * inferred from the rows.
 */
export function commitWarnings(batch: RateImportBatch | undefined): string[] {
  if (!batch) return [];

  const warnings: string[] = [];

  if (batch.errorCount > 0) {
    warnings.push(
      `${batch.errorCount} of ${batch.rowCount} rows could not be read and will not be imported. ` +
        `No lane is shown as stopping until they are fixed, because a row that would not read ` +
        `named no lane.`,
    );
  }

  const unmapped = batch.unmappedHeaders ?? [];
  if (unmapped.length > 0) {
    warnings.push(`Nothing was read from these columns: ${unmapped.join(", ")}.`);
  }

  const duplicates = batch.summary?.duplicate ?? 0;
  if (duplicates > 0) {
    warnings.push(
      `${duplicates} lanes are listed more than once. Only the first of each will be imported.`,
    );
  }

  const removed = batch.summary?.removed ?? 0;
  if (removed > 0) {
    warnings.push(`${removed} lanes in the agreement are not in this sheet and will stop pricing.`);
  }

  return warnings;
}
