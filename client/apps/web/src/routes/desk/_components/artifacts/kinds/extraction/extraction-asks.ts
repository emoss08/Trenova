import { formatList } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { rowReason, stopTitle } from "./extraction-words";
import type { CheckItem, FieldRow, ReviewTally } from "./extraction-groups";

/** What Check asks about: each field to look at again, and each one to look for. */
export type CheckNames = {
  /** Each field to check, with why, as the agent is told. */
  check: string[];
  /** Each field the document did not have. */
  missing: string[];
  /** The names alone, as the card shows them. */
  shown: { check: string[]; missing: string[] };
};

function itemName(item: CheckItem, t: TranslateFn): string {
  return item.kind === "stop" ? stopTitle(item.stop, t) : item.row.label;
}

function rowWithReason(row: FieldRow, t: TranslateFn): string {
  const reason = rowReason(row, t);
  return reason === null ? row.label : t("{0} ({1})", row.label, reason);
}

function itemWithReason(item: CheckItem, t: TranslateFn): string {
  if (item.kind === "field") return rowWithReason(item.row, t);
  const doubtful = item.stop.rows.filter((row) => row.state === "doubtful");
  return t(
    "{0}: {1}",
    stopTitle(item.stop, t),
    formatList(doubtful.map((row) => rowWithReason(row, t))),
  );
}

export function checkNames(tally: ReviewTally, t: TranslateFn): CheckNames {
  return {
    check: tally.check.map((item) => itemWithReason(item, t)),
    missing: tally.missing.map((row) => row.label),
    shown: {
      check: tally.check.map((item) => itemName(item, t)),
      missing: tally.missing.map((row) => row.label),
    },
  };
}

/** How many fields Check names. */
export function checkCount(tally: ReviewTally): number {
  return tally.check.length + tally.missing.length;
}

type FileRef = { fileName: string; documentId: string };

/**
 * The question Check sends: the fields to look at again with why, then those to look for, on
 * this file. It names the document's id so the agent reads the right draft.
 */
export function checkQuestion(file: FileRef, names: CheckNames, t: TranslateFn): string {
  const check = formatList(names.check);
  const missing = formatList(names.missing);
  if (names.check.length > 0 && names.missing.length > 0) {
    return t(
      "On {0} (document {1}), check these fields against the document and correct what was misread: {2}. Then look for what was not found: {3}.",
      file.fileName,
      file.documentId,
      check,
      missing,
    );
  }
  if (names.check.length > 0) {
    return t(
      "On {0} (document {1}), check these fields against the document and correct what was misread: {2}.",
      file.fileName,
      file.documentId,
      check,
    );
  }
  return t(
    "On {0} (document {1}), look for what was not found: {2}.",
    file.fileName,
    file.documentId,
    missing,
  );
}

/** What the card says Check will do, above the buttons. */
export function checkSummary(names: CheckNames, t: TranslateFn): string {
  const check = formatList(names.shown.check);
  const missing = formatList(names.shown.missing);
  if (names.check.length > 0 && names.missing.length > 0) {
    return t("Check asks the agent to look again at {0}, and to look for {1}.", check, missing);
  }
  if (names.check.length > 0) return t("Check asks the agent to look again at {0}.", check);
  return t("Check asks the agent to look for {0}.", missing);
}

/** The question Create sends; the agent drafts the shipment and it waits for approval. */
export function createQuestion(file: FileRef, t: TranslateFn): string {
  return t(
    "Create a shipment from {0} (document {1}), using what was read from it.",
    file.fileName,
    file.documentId,
  );
}
