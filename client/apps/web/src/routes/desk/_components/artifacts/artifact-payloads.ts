import type { ReportPreviewColumn } from "@/lib/graphql/reports";
import type { AssistantArtifact } from "@/types/assistant";
import { toTitleCase } from "@trenova/shared/lib/utils";

/*
 * The server stores each artifact's payload in the shape the tool published
 * it. These readers turn that into what a renderer needs and refuse to
 * guess: a value that is not what the kind promises reads as absent, never
 * as a crash in the pane.
 */

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringOf(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function numberOf(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function listOf(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

export type ReportPreviewArtifact = {
  name: string;
  dataset: string;
  columns: ReportPreviewColumn[];
  rows: unknown[][];
  totals: unknown[] | null;
  rowCount: number;
  truncated: boolean;
};

function previewColumn(value: unknown): ReportPreviewColumn | null {
  if (!isRecord(value)) {
    return null;
  }
  const id = stringOf(value.id);
  const label = stringOf(value.label);
  if (id === "" && label === "") {
    return null;
  }

  return {
    id: id || label,
    label: label || id,
    type: stringOf(value.type) || "string",
    format: stringOf(value.format) || null,
  } as ReportPreviewColumn;
}

/** Rows arrive keyed by column label; the grid wants them positional. */
export function reportPreviewFrom(artifact: AssistantArtifact): ReportPreviewArtifact {
  const payload = artifact.payload;
  const columns = listOf(payload.columns)
    .map(previewColumn)
    .filter((column): column is ReportPreviewColumn => column !== null);
  const rows = listOf(payload.rows)
    .filter(isRecord)
    .map((row) => columns.map((column) => row[column.label] ?? row[column.id] ?? null));
  const totalsRow = payload.totals;
  const totals = isRecord(totalsRow)
    ? columns.map((column) => totalsRow[column.label] ?? totalsRow[column.id] ?? null)
    : null;

  return {
    name: stringOf(payload.name),
    dataset: stringOf(payload.dataset),
    columns,
    rows,
    totals: totals && totals.some((value) => value !== null) ? totals : null,
    rowCount: numberOf(payload.rowCount) || rows.length,
    truncated: payload.truncated === true,
  };
}

export type TableViewArtifact = {
  tool: string;
  entity: string;
  searchedFor: string[];
  columns: ReportPreviewColumn[];
  rows: unknown[][];
  rowCount: number;
  truncated: boolean;
};

/**
 * A list or search result as a table.
 *
 * The columns come from the server in the order the row projection declares
 * them, because a JSON object has no order and the pane would otherwise draw
 * "pro number, customer, status" in whatever order the payload happened to
 * serialize. The rows are keyed objects, so they are read positionally
 * against those columns; a row missing a column reads as an empty cell, never
 * as a shifted one.
 */
export function tableViewFrom(artifact: AssistantArtifact): TableViewArtifact {
  const payload = artifact.payload;
  const columns = listOf(payload.columns)
    .filter((name): name is string => typeof name === "string" && name !== "")
    .map(
      (name) =>
        ({
          id: name,
          label: toTitleCase(name),
          type: "string",
          format: null,
        }) as ReportPreviewColumn,
    );
  const rows = listOf(payload.rows)
    .filter(isRecord)
    .map((row) => columns.map((column) => cell(row[column.id])));
  const rowCount = numberOf(payload.rowCount) || rows.length;

  return {
    tool: stringOf(payload.tool),
    entity: stringOf(payload.entity),
    searchedFor: listOf(payload.searchedFor).filter(
      (term): term is string => typeof term === "string" && term !== "",
    ),
    columns,
    rows,
    rowCount,
    // The count is what the search found; the rows are what fitted in the
    // payload. A table that says 400 above 200 rows has to say why.
    truncated: rowCount > rows.length,
  };
}

/**
 * One value as the grid can draw it. A nested object or list is a cell the
 * grid has no column layout for, so it is rendered as its JSON rather than
 * as "[object Object]".
 */
function cell(value: unknown): unknown {
  if (value === undefined) return null;
  if (value === null || typeof value !== "object") return value;

  return JSON.stringify(value);
}

/** A described view, as something to open rather than rows to read. */
export type ComposedViewArtifact = {
  entity: string;
  path: string;
  explanation: string;
  terms: string[];
  filterCount: number;
  unresolved: { phrase: string; reason: string }[];
};

export function composedViewFrom(artifact: AssistantArtifact): ComposedViewArtifact | null {
  const path = stringOf(artifact.payload.path);
  if (path === "") {
    return null;
  }

  return {
    entity: stringOf(artifact.payload.entity),
    path,
    explanation: stringOf(artifact.payload.explanation),
    terms: listOf(artifact.payload.terms).filter(
      (term): term is string => typeof term === "string" && term !== "",
    ),
    filterCount: numberOf(artifact.payload.filterCount),
    unresolved: listOf(artifact.payload.unresolved)
      .filter(isRecord)
      .map((entry) => ({ phrase: stringOf(entry.phrase), reason: stringOf(entry.reason) }))
      .filter((entry) => entry.phrase !== ""),
  };
}

export type RateComponent = {
  label: string;
  basis: string;
  amount: string;
  runningTotal: string;
};

export type RateGuardrail = { kind: string; bound: string; raw: string; result: string };

export type RateExplanationArtifact = {
  shipmentId: string;
  side: string;
  currency: string;
  winner: { agreementCode: string; agreementName: string; ruleLabel: string } | null;
  tieBreak: string;
  rejected: { agreementCode: string; ruleLabel: string; reason: string; detail: string }[];
  components: RateComponent[];
  guardrails: RateGuardrail[];
  totals: { linehaul: string; fuel: string; accessorial: string; total: string };
  warnings: string[];
};

/**
 * A price as a ledger.
 *
 * Every figure is passed through as the engine wrote it — amounts are decimal
 * strings on the wire and stay strings here, because reading money into a
 * JavaScript number is how a cent goes missing between the explanation and
 * the invoice it is meant to match.
 */
export function rateExplanationFrom(artifact: AssistantArtifact): RateExplanationArtifact {
  const payload = artifact.payload;
  const winner = isRecord(payload.winner) ? payload.winner : null;

  return {
    shipmentId: stringOf(payload.shipmentId),
    side: stringOf(payload.side),
    currency: stringOf(payload.currency),
    winner:
      winner === null
        ? null
        : {
            agreementCode: stringOf(winner.agreementCode),
            agreementName: stringOf(winner.agreementName),
            ruleLabel: stringOf(winner.ruleLabel),
          },
    tieBreak: stringOf(payload.tieBreak),
    rejected: listOf(payload.rejected)
      .filter(isRecord)
      .map((entry) => ({
        agreementCode: stringOf(entry.agreementCode),
        ruleLabel: stringOf(entry.ruleLabel),
        reason: stringOf(entry.reason),
        detail: stringOf(entry.detail),
      })),
    components: listOf(payload.components)
      .filter(isRecord)
      .map((entry) => ({
        label: stringOf(entry.label),
        basis: stringOf(entry.basis),
        amount: amountOf(entry.amount),
        runningTotal: amountOf(entry.runningTotal),
      })),
    guardrails: listOf(payload.guardrails)
      .filter(isRecord)
      .map((entry) => ({
        kind: stringOf(entry.kind),
        bound: amountOf(entry.bound),
        raw: amountOf(entry.raw),
        result: amountOf(entry.result),
      })),
    totals: totalsOf(payload.totals),
    warnings: listOf(payload.warnings).filter(
      (warning): warning is string => typeof warning === "string" && warning !== "",
    ),
  };
}

/** Money stays as it arrived. A number would round it. */
function amountOf(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);

  return "";
}

function totalsOf(value: unknown) {
  const totals = isRecord(value) ? value : {};

  return {
    linehaul: amountOf(totals.linehaul),
    fuel: amountOf(totals.fuel),
    accessorial: amountOf(totals.accessorial),
    total: amountOf(totals.total),
  };
}

export type ReportRunArtifact = {
  runId: string;
  reportKey: string;
  reportName: string;
};

export function reportRunFrom(artifact: AssistantArtifact): ReportRunArtifact | null {
  const runId = stringOf(artifact.payload.runId);
  if (runId === "") {
    return null;
  }

  return {
    runId,
    reportKey: stringOf(artifact.payload.reportKey),
    reportName: stringOf(artifact.payload.reportName).trim(),
  };
}

export type EmailDraftArtifact = {
  tool: string;
  subject: string;
  body: string;
  to: string[];
  rationale: string;
};

export function emailDraftFrom(artifact: AssistantArtifact): EmailDraftArtifact {
  const payload = artifact.payload;
  const to = Array.isArray(payload.to)
    ? payload.to.filter((value): value is string => typeof value === "string" && value !== "")
    : typeof payload.to === "string" && payload.to !== ""
      ? [payload.to]
      : [];

  return {
    tool: stringOf(payload.tool),
    subject: stringOf(payload.subject),
    body: stringOf(payload.body),
    to,
    rationale: stringOf(payload.rationale),
  };
}

export type EntityFact = { key: string; value: string };

export type EntityCardArtifact = {
  entity: string;
  id: string;
  facts: EntityFact[];
  record: Record<string, unknown>;
};

/** Keys every record carries that say nothing about it. */
const STRUCTURAL_KEYS = new Set(["id", "organizationId", "businessUnitId", "version"]);

function scalar(value: unknown): string | null {
  if (value === null || value === undefined || value === "") return null;
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return null;
}

/**
 * A record as a handful of labelled values, which is how a person reads one.
 * Nested records and lists are left to the raw view; the id and tenancy are
 * structure, not facts.
 */
export function entityCardFrom(artifact: AssistantArtifact): EntityCardArtifact {
  const record = isRecord(artifact.payload.record) ? artifact.payload.record : {};
  const facts: EntityFact[] = [];
  for (const [key, value] of Object.entries(record)) {
    if (STRUCTURAL_KEYS.has(key)) continue;
    const text = scalar(value);
    if (text !== null) {
      facts.push({ key, value: text });
    }
  }

  return {
    entity: stringOf(artifact.payload.entity),
    id: stringOf(record.id),
    facts,
    record,
  };
}

export type PlanStepArtifact = {
  step: number;
  proposalId: string;
  toolName: string;
  rationale: string;
};

export type PlanArtifact = {
  title: string;
  summary: string;
  stepCount: number;
  steps: PlanStepArtifact[];
};

export function planFrom(artifact: AssistantArtifact): PlanArtifact {
  const payload = artifact.payload;
  const steps = listOf(payload.steps)
    .filter(isRecord)
    .map((step) => ({
      step: numberOf(step.step),
      proposalId: stringOf(step.proposalId),
      toolName: stringOf(step.toolName),
      rationale: stringOf(step.rationale),
    }))
    .sort((a, b) => a.step - b.step);

  return {
    title: stringOf(payload.title) || artifact.title,
    summary: stringOf(payload.summary),
    stepCount: numberOf(payload.stepCount) || steps.length,
    steps,
  };
}

export type RunDiffSideArtifact = {
  runId: string;
  reportName: string;
  generatedAt: number;
  rowCount: number;
  truncated: boolean;
};

export type RunDiffMeasureArtifact = {
  column: string;
  label: string;
  before: string;
  after: string;
  delta: string;
};

export type RunDiffChangeArtifact = {
  kind: string;
  key: string;
  keyValues: string[];
  measures: RunDiffMeasureArtifact[];
};

export type RunDiffArtifactPayload = {
  before: RunDiffSideArtifact;
  after: RunDiffSideArtifact;
  keys: string[];
  measures: string[];
  counts: { added: number; removed: number; changed: number; unchanged: number; duplicate: number };
  changes: RunDiffChangeArtifact[];
  totals: RunDiffMeasureArtifact[];
  truncated: boolean;
  note: string;
};

function diffSide(value: unknown): RunDiffSideArtifact {
  const side = isRecord(value) ? value : {};

  return {
    runId: stringOf(side.runId),
    reportName: stringOf(side.reportName),
    generatedAt: numberOf(side.generatedAt),
    rowCount: numberOf(side.rowCount),
    truncated: side.truncated === true,
  };
}

function diffMeasure(value: unknown): RunDiffMeasureArtifact | null {
  if (!isRecord(value)) {
    return null;
  }
  const column = stringOf(value.column);
  if (column === "") {
    return null;
  }

  return {
    column,
    label: stringOf(value.label) || column,
    before: stringOf(value.before),
    after: stringOf(value.after),
    delta: stringOf(value.delta),
  };
}

function diffMeasures(value: unknown): RunDiffMeasureArtifact[] {
  return listOf(value)
    .map(diffMeasure)
    .filter((measure): measure is RunDiffMeasureArtifact => measure !== null);
}

/**
 * What moved between two runs.
 *
 * The counts come off the server's own summary rather than from the length of
 * the change list: the list is bounded so an answer stays readable, and
 * counting it instead would report "3 changed" for a report where three
 * hundred did.
 */
export function runDiffFrom(artifact: AssistantArtifact): RunDiffArtifactPayload {
  const payload = artifact.payload;
  const summary = isRecord(payload.summary) ? payload.summary : {};

  const changes = listOf(payload.changes)
    .filter(isRecord)
    .map((change) => ({
      kind: stringOf(change.kind),
      key: stringOf(change.key),
      keyValues: listOf(change.keyValues).map(stringOf),
      measures: diffMeasures(change.measures),
    }));

  return {
    before: diffSide(payload.before),
    after: diffSide(payload.after),
    keys: listOf(payload.keys).map(stringOf),
    measures: listOf(payload.measures).map(stringOf),
    counts: {
      added: numberOf(summary.added),
      removed: numberOf(summary.removed),
      changed: numberOf(summary.changed),
      unchanged: numberOf(summary.unchanged),
      duplicate: numberOf(summary.duplicate),
    },
    changes,
    totals: diffMeasures(payload.totals),
    truncated: payload.truncated === true,
    note: stringOf(payload.note),
  };
}
