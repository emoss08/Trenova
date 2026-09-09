import type { FuelPurchaseImportBatch } from "@/lib/graphql/fuel-purchase-import";
import type { FuelPurchaseImportRowStatus } from "@trenova/graphql/generated/graphql";
import { formatCurrency, pluralize } from "@trenova/shared/lib/utils";

export type ImportStep = "setup" | "review" | "done";

export type ImportRowFilter = "all" | "new" | "duplicates" | "errors";

export const IMPORT_ROW_FILTER_STATUSES: Record<
  ImportRowFilter,
  FuelPurchaseImportRowStatus[] | undefined
> = {
  all: undefined,
  new: ["New"],
  duplicates: ["DuplicateInFile", "AlreadyImported"],
  errors: ["Error"],
};

export const IMPORT_ROW_FILTER_LABELS: Record<ImportRowFilter, string> = {
  all: "All",
  new: "New",
  duplicates: "Duplicates",
  errors: "Errors",
};

export const IMPORT_ROW_STATUS_LABELS: Record<FuelPurchaseImportRowStatus, string> = {
  New: "New",
  DuplicateInFile: "Duplicate in file",
  AlreadyImported: "Already imported",
  Error: "Error",
  Committed: "Committed",
  Skipped: "Skipped",
};

export function importStep(batch: FuelPurchaseImportBatch | undefined): ImportStep {
  if (!batch) return "setup";
  switch (batch.status) {
    case "Pending":
      return "setup";
    case "Parsed":
    case "Failed":
      return "review";
    case "Committed":
    case "Discarded":
      return "done";
    default:
      return "setup";
  }
}

export function canCommitImport(batch: FuelPurchaseImportBatch | undefined): boolean {
  return batch?.status === "Parsed" && (batch.summary?.newCount ?? 0) > 0;
}

function count(n: number, noun: string): string {
  return `${n} ${pluralize(noun, n)}`;
}

export function commitLabel(batch: FuelPurchaseImportBatch | undefined): string {
  const newCount = batch?.summary?.newCount ?? 0;
  return newCount > 0 ? `Import ${count(newCount, "purchase")}` : "Import purchases";
}

export function importHeadline(batch: FuelPurchaseImportBatch | undefined): string {
  if (!batch) return "Waiting for the statement to be read.";
  switch (batch.status) {
    case "Pending":
      return "Waiting for the statement to be read.";
    case "Failed":
      return "The statement could not be read.";
    case "Committed":
      return `Recorded ${count(batch.committedCount, "purchase")}.`;
    case "Discarded":
      return "This import was discarded; nothing was recorded.";
    case "Parsed": {
      const summary = batch.summary;
      if (!summary || summary.newCount === 0) {
        return "Nothing in this statement is new: every row is a duplicate, already on file, or could not be read.";
      }
      const amount = formatCurrency(Number(summary.totalAmount), batch.defaultCurrency);
      return `Importing would record ${count(summary.newCount, "purchase")} for ${summary.totalGallons} gallons and ${amount}.`;
    }
    default:
      return "";
  }
}

export function importWarnings(batch: FuelPurchaseImportBatch | undefined): string[] {
  if (!batch) return [];
  const warnings: string[] = [];
  if (batch.unmappedHeaders.length > 0) {
    warnings.push(`Columns ignored: ${batch.unmappedHeaders.join(", ")}.`);
  }
  const summary = batch.summary;
  if (!summary) return warnings;
  if (summary.duplicateInFileCount > 0) {
    const n = summary.duplicateInFileCount;
    warnings.push(
      `${count(n, "row")} ${n === 1 ? "repeats" : "repeat"} a reference earlier in this statement and will be skipped.`,
    );
  }
  if (summary.alreadyImportedCount > 0) {
    const n = summary.alreadyImportedCount;
    warnings.push(
      `${count(n, "row")} ${n === 1 ? "matches" : "match"} a purchase already on file and will be skipped.`,
    );
  }
  if (summary.errorCount > 0) {
    warnings.push(`${count(summary.errorCount, "row")} could not be read and will be skipped.`);
  }
  return warnings;
}

export function needsDiscardConfirmation(batch: FuelPurchaseImportBatch | undefined): boolean {
  return batch?.status === "Parsed";
}

export function discardImportNotice(batch: FuelPurchaseImportBatch): {
  title: string;
  description: string;
} {
  const source = batch.fileName ? ` from ${batch.fileName}` : "";
  return {
    title: "Discard this import?",
    description: `The ${count(batch.rowCount, "row")} read${source} will be thrown away. Nothing has been recorded, and the statement can be uploaded again.`,
  };
}
