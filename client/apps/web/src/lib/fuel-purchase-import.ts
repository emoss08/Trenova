import { translate } from "@trenova/shared/i18n/runtime";
import type { FuelPurchaseImportBatch } from "@/lib/graphql/fuel-purchase-import";
import type { FuelPurchaseImportRowStatus } from "@trenova/graphql/generated/graphql";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { defineLabels } from "@trenova/shared/i18n/labels";

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

export const IMPORT_ROW_FILTER_LABELS: Record<ImportRowFilter, string> = defineLabels({
  all: "All",
  new: "New",
  duplicates: "Duplicates",
  errors: "Errors",
});

export const IMPORT_ROW_STATUS_LABELS: Record<FuelPurchaseImportRowStatus, string> = defineLabels({
  New: "New",
  DuplicateInFile: "Duplicate in file",
  AlreadyImported: "Already imported",
  Error: "Error",
  Committed: "Committed",
  Skipped: "Skipped",
});

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

export function commitLabel(batch: FuelPurchaseImportBatch | undefined): string {
  const newCount = batch?.summary?.newCount ?? 0;
  return newCount > 0
    ? translate("{0, plural, one {Import # purchase} other {Import # purchases}}", newCount)
    : translate("Import purchases");
}

export function importHeadline(batch: FuelPurchaseImportBatch | undefined): string {
  if (!batch) return translate("Waiting for the statement to be read.");
  switch (batch.status) {
    case "Pending":
      return translate("Waiting for the statement to be read.");
    case "Failed":
      return translate("The statement could not be read.");
    case "Committed":
      return translate(
        "{0, plural, one {Recorded # purchase.} other {Recorded # purchases.}}",
        batch.committedCount,
      );
    case "Discarded":
      return translate("This import was discarded; nothing was recorded.");
    case "Parsed": {
      const summary = batch.summary;
      if (!summary || summary.newCount === 0) {
        return translate(
          "Nothing in this statement is new: every row is a duplicate, already on file, or could not be read.",
        );
      }
      const amount = formatCurrency(Number(summary.totalAmount), batch.defaultCurrency);
      return translate(
        "Importing would record {0, plural, one {# purchase} other {# purchases}} for {1} gallons and {2}.",
        summary.newCount,
        summary.totalGallons,
        amount,
      );
    }
    default:
      return "";
  }
}

export function importWarnings(batch: FuelPurchaseImportBatch | undefined): string[] {
  if (!batch) return [];
  const warnings: string[] = [];
  if (batch.unmappedHeaders.length > 0) {
    warnings.push(translate("Columns ignored: {0}.", batch.unmappedHeaders.join(", ")));
  }
  const summary = batch.summary;
  if (!summary) return warnings;
  if (summary.duplicateInFileCount > 0) {
    warnings.push(
      translate(
        "{0, plural, one {# row repeats a reference earlier in this statement and will be skipped.} other {# rows repeat a reference earlier in this statement and will be skipped.}}",
        summary.duplicateInFileCount,
      ),
    );
  }
  if (summary.alreadyImportedCount > 0) {
    warnings.push(
      translate(
        "{0, plural, one {# row matches a purchase already on file and will be skipped.} other {# rows match a purchase already on file and will be skipped.}}",
        summary.alreadyImportedCount,
      ),
    );
  }
  if (summary.errorCount > 0) {
    warnings.push(
      translate(
        "{0, plural, one {# row could not be read and will be skipped.} other {# rows could not be read and will be skipped.}}",
        summary.errorCount,
      ),
    );
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
  return {
    title: translate("Discard this import?"),
    description: batch.fileName
      ? translate(
          "The {0, plural, one {# row} other {# rows}} read from {1} will be thrown away. Nothing has been recorded, and the statement can be uploaded again.",
          batch.rowCount,
          batch.fileName,
        )
      : translate(
          "The {0, plural, one {# row} other {# rows}} read will be thrown away. Nothing has been recorded, and the statement can be uploaded again.",
          batch.rowCount,
        ),
  };
}
