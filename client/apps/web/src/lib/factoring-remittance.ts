import type {
  FactoringRemittanceColumnInput,
  FactoringRemittanceField,
  FactoringRemittanceLineKind,
  FactoringRemittanceMatchStatus,
  FactoringRemittanceStatus,
} from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";

export const FACTORING_REMITTANCE_STATUS_LABELS = defineLabels({
  Pending: "Reading",
  NeedsMapping: "Needs mapping",
  Parsed: "Ready to apply",
  Failed: "Could not read",
  PartiallyApplied: "Partly applied",
  Applied: "Applied",
  Discarded: "Discarded",
});

const STATUS_PHASES: Record<FactoringRemittanceStatus, StatusPhase> = {
  Pending: "queued",
  NeedsMapping: "attention",
  Parsed: "awaiting",
  Failed: "failed",
  PartiallyApplied: "attention",
  Applied: "complete",
  Discarded: "closed",
};

export function factoringRemittancePhase(status: FactoringRemittanceStatus): StatusPhase {
  return STATUS_PHASES[status];
}

export const FACTORING_REMITTANCE_LINE_KIND_LABELS = defineLabels({
  Funding: "Funding",
  Rejection: "Rejection",
  ReserveRelease: "Reserve release",
  Chargeback: "Chargeback",
  Other: "Adjustment",
});

/** The kinds a whole report can be of; an adjustment is a line, never a report. */
export const FACTORING_REMITTANCE_REPORT_KINDS: readonly FactoringRemittanceLineKind[] = [
  "Funding",
  "ReserveRelease",
  "Rejection",
  "Chargeback",
];

export const FACTORING_REMITTANCE_MATCH_STATUS_LABELS = defineLabels({
  Matched: "Matched",
  Suggested: "Suggested",
  Unmatched: "No match",
  Ignored: "Ignored",
  Error: "Unreadable",
  Applied: "Booked",
});

const MATCH_PHASES: Record<FactoringRemittanceMatchStatus, StatusPhase> = {
  Matched: "awaiting",
  Suggested: "attention",
  Unmatched: "attention",
  Ignored: "closed",
  Error: "failed",
  Applied: "complete",
};

export function factoringRemittanceMatchPhase(status: FactoringRemittanceMatchStatus): StatusPhase {
  return MATCH_PHASES[status];
}

export const FACTORING_REMITTANCE_MATCH_REASON_LABELS = defineLabels({
  InvoiceAmountDate: "Invoice, amount and date agree",
  InvoiceAmount: "Invoice and amount agree",
  InvoiceNearAmount: "Invoice agrees, amount is close",
  PartialInvoiceAmount: "Invoice partly agrees, amount agrees",
  InvoiceOnly: "Invoice agrees, amount differs",
  PartialInvoiceNearAmount: "Invoice partly agrees, amount is close",
  AmountOnly: "Only the amount agrees",
  NearAmountOnly: "Only a close amount",
  ClaimedByAnotherLine: "Another line already matched it",
  ChosenByPerson: "Chosen by a person",
  NoOpenItem: "No open invoice on the factor's schedules",
  NotBookedFromRemittance: "An adjustment; book it by hand",
});

export const FACTORING_REMITTANCE_FIELD_LABELS = defineLabels({
  InvoiceNumber: "Invoice number",
  FactorReference: "Factor reference",
  Debtor: "Debtor",
  Kind: "Transaction type",
  Face: "Invoice amount",
  Advance: "Advance",
  Fee: "Fee",
  Reserve: "Reserve",
  Cash: "Amount paid",
  Date: "Date",
});

/** Every field a column can hold, in the order the mapping step lists them. */
export const FACTORING_REMITTANCE_FIELDS: readonly FactoringRemittanceField[] = [
  "InvoiceNumber",
  "FactorReference",
  "Debtor",
  "Kind",
  "Face",
  "Advance",
  "Fee",
  "Reserve",
  "Cash",
  "Date",
];

const AMOUNT_FIELDS: readonly FactoringRemittanceField[] = ["Cash", "Advance", "Face", "Reserve"];

/** A column choice per field; null means the file has no such column. */
export type RemittanceMapping = Partial<Record<FactoringRemittanceField, number | null>>;

export function mappingFromColumns(
  columns: ReadonlyArray<{ field: FactoringRemittanceField; column: number }>,
): RemittanceMapping {
  const mapping: RemittanceMapping = {};
  for (const { field, column } of columns) {
    mapping[field] = column;
  }
  return mapping;
}

export function mappingToColumns(mapping: RemittanceMapping): FactoringRemittanceColumnInput[] {
  return FACTORING_REMITTANCE_FIELDS.flatMap((field) => {
    const column = mapping[field];
    return column === null || column === undefined ? [] : [{ field, column }];
  });
}

export type MappingProblem =
  | { kind: "reference" }
  | { kind: "amount" }
  | { kind: "duplicate"; fields: [FactoringRemittanceField, FactoringRemittanceField] };

/**
 * What the server's mapping check would refuse, in the same order: a column for
 * the invoice or the factor's reference, a column for an amount, and no column
 * read by two fields.
 */
export function mappingProblems(mapping: RemittanceMapping): MappingProblem[] {
  const problems: MappingProblem[] = [];
  const has = (field: FactoringRemittanceField) =>
    mapping[field] !== null && mapping[field] !== undefined;
  if (!has("InvoiceNumber") && !has("FactorReference")) {
    problems.push({ kind: "reference" });
  }
  if (!AMOUNT_FIELDS.some(has)) {
    problems.push({ kind: "amount" });
  }
  const seen = new Map<number, FactoringRemittanceField>();
  for (const field of FACTORING_REMITTANCE_FIELDS) {
    const column = mapping[field];
    if (column === null || column === undefined) continue;
    const other = seen.get(column);
    if (other) {
      problems.push({ kind: "duplicate", fields: [other, field] });
      continue;
    }
    seen.set(column, field);
  }
  return problems;
}

export function isRemittanceException(status: FactoringRemittanceMatchStatus): boolean {
  return status === "Suggested" || status === "Unmatched" || status === "Error";
}

export function canEditRemittanceLines(status: FactoringRemittanceStatus): boolean {
  return status === "Parsed" || status === "PartiallyApplied";
}

export function canStageRemittance(status: FactoringRemittanceStatus): boolean {
  return (
    status === "Pending" || status === "NeedsMapping" || status === "Parsed" || status === "Failed"
  );
}

export function canApplyRemittance(remittance: {
  status: FactoringRemittanceStatus;
  matchedCount: number;
}): boolean {
  return canEditRemittanceLines(remittance.status) && remittance.matchedCount > 0;
}

export function canDiscardRemittance(status: FactoringRemittanceStatus): boolean {
  return canStageRemittance(status);
}

export type RemittanceStep = "upload" | "reading" | "mapping" | "review" | "done" | "failed";

/** Which part of the remittance page a remittance in this state shows. */
export function remittanceStep(remittance: {
  status: FactoringRemittanceStatus;
  documentId?: string | null;
}): RemittanceStep {
  switch (remittance.status) {
    case "Pending":
      return remittance.documentId ? "reading" : "upload";
    case "NeedsMapping":
      return "mapping";
    case "Parsed":
    case "PartiallyApplied":
      return "review";
    case "Applied":
    case "Discarded":
      return "done";
    case "Failed":
      return "failed";
  }
}

/** The file types a factor's remittance arrives as. */
export const FACTORING_REMITTANCE_ACCEPT = {
  "text/csv": [".csv"],
  "text/plain": [".txt"],
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
  "application/pdf": [".pdf"],
} as const;

/** The files a remittance is read from, as a file picker's `accept` list. */
export const FACTORING_REMITTANCE_FILE_TYPES = ".csv,.txt,.xlsx,.pdf";

/** The largest remittance file the server reads: 32 MiB. */
export const FACTORING_REMITTANCE_MAX_BYTES = 32 * 1024 * 1024;

export function isPdfRemittance(fileName: string): boolean {
  return fileName.trim().toLowerCase().endsWith(".pdf");
}

/** A score as the matcher gives it, out of 100. */
export function matchScoreLabel(score: number): string {
  return `${Math.max(0, Math.min(100, Math.round(score)))}%`;
}
