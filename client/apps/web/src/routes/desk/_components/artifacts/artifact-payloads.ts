import {
  RECORD_ID_KEY,
  formatDisplayValue,
  projectColumns,
  projectRecord,
  projectValue,
  readDisplayColumns,
  readDisplayFields,
  sortKey,
  type DisplayColumn,
  type DisplayField,
} from "@/components/assistant/readable-values";
import { isRecordEntityType, recordPath, type RecordEntityType } from "@/config/record-links";
import { isAppPath } from "@/lib/app-path";
import type { ReportPreviewColumn } from "@/lib/graphql/reports";
import type { AssistantArtifact } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

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

/** One row of a list result: its values by column, and where its record opens. */
export type TableViewRow = {
  key: string;
  values: Record<string, unknown>;
  /** The record's page, built from the registry; empty when it has none. */
  path: string;
};

export type TableViewArtifact = {
  tool: string;
  entity: string;
  searchedFor: string[];
  columns: DisplayColumn[];
  rows: TableViewRow[];
  rowCount: number;
  truncated: boolean;
  /** How many tool calls the server read this table together from; zero for one search's result. */
  calls: number;
};

function singular(plural: string): string {
  if (plural.endsWith("ices")) return `${plural.slice(0, -4)}ix`;
  if (plural.endsWith("ies")) return `${plural.slice(0, -3)}y`;
  if (plural.endsWith("sses")) return plural.slice(0, -2);
  if (plural.endsWith("s")) return plural.slice(0, -1);
  return plural;
}

/**
 * The kind of record a table's rows are, when they have a page to open. A
 * projected payload says so itself, and says nothing when they have none; one
 * stored before the projection is read from the list it came from.
 */
function tableRecordEntity(payload: Record<string, unknown>): RecordEntityType | null {
  const declared = stringOf(payload.recordEntity);
  if (declared !== "" || payload.display !== undefined) {
    return isRecordEntityType(declared) ? declared : null;
  }
  const entity = stringOf(payload.entity);
  for (const candidate of [entity, singular(entity)]) {
    if (candidate !== "" && isRecordEntityType(candidate)) {
      return candidate;
    }
  }

  return null;
}

/**
 * A list or search result as a table a person reads.
 *
 * The server stores the projection — the columns a person can reason with,
 * each typed, and rows holding only those — so a new payload is read as it
 * is. One stored before that holds the tool's own columns and rows, ids and
 * JSON and all, and is projected here by the same rules, so an old table
 * reads the same as a new one. Either way a cell is read against its column,
 * so a row missing a value leaves a hole rather than shifting its neighbours.
 *
 * A row's record id is kept only to build the link to its page, from the
 * record-link registry, and is never a column.
 */
export function tableViewFrom(artifact: AssistantArtifact): TableViewArtifact {
  const payload = artifact.payload;
  const raw = listOf(payload.rows).filter(isRecord);
  const columns =
    readDisplayColumns(payload.columns) ??
    projectColumns(
      listOf(payload.columns).filter(
        (name): name is string => typeof name === "string" && name !== "",
      ),
      raw,
    );
  const entity = tableRecordEntity(payload);
  const rows = raw.map((row, index) => {
    const values: Record<string, unknown> = {};
    for (const column of columns) {
      const value = projectValue(column.type, row[column.key]);
      if (value !== undefined) {
        values[column.key] = value;
      }
    }
    const id = stringOf(row[RECORD_ID_KEY]);

    return {
      key: String(index),
      values,
      path: entity !== null && id !== "" ? recordPath(entity, id) : "",
    };
  });
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
    calls: bunchedCallCount(payload),
  };
}

/**
 * How many calls a table the server read together was made from. The server
 * folds one record after another into one table and lists the calls it
 * folded; a table it did not fold was one search and counts no calls.
 */
function bunchedCallCount(payload: Record<string, unknown>): number {
  if (payload.bunched !== true) {
    return 0;
  }
  const calls = listOf(payload.calls).filter(
    (call): call is string => typeof call === "string" && call !== "",
  ).length;

  return calls > 0 ? calls : numberOf(payload.rowCount) || listOf(payload.rows).length;
}

/**
 * What the provenance line adds for an artifact whose title does not say how
 * it was made: a table read together from several calls says so, because
 * "Billing queue item (11)" alone reads as one search's result.
 */
export function artifactProvenanceNote(artifact: AssistantArtifact, t: TranslateFn): string {
  if (artifact.kind !== "table_view") {
    return "";
  }
  const calls = bunchedCallCount(artifact.payload);

  return calls > 0 ? t("Read together from {0, plural, one {# call} other {# calls}}", calls) : "";
}

/**
 * The rows a few typed letters keep: any cell, read as a person reads it, that
 * contains them. A status matches its words and its code, so "transit" and
 * "InTransit" both find a shipment on the road.
 */
export function filterTableRows(
  rows: readonly TableViewRow[],
  columns: readonly DisplayColumn[],
  query: string,
  t: TranslateFn,
): TableViewRow[] {
  const needle = query.trim().toLocaleLowerCase();
  if (needle === "") {
    return [...rows];
  }

  return rows.filter((row) =>
    columns.some((column) => {
      const value = row.values[column.key];
      if (value === undefined || value === null) {
        return false;
      }
      if (formatDisplayValue(column.type, value, t).toLocaleLowerCase().includes(needle)) {
        return true;
      }

      return typeof value === "string" && value.toLocaleLowerCase().includes(needle);
    }),
  );
}

export type TableSort = { key: string; direction: "asc" | "desc" };

/**
 * Rows in a column's order: figures and instants by value, a status by how
 * much it needs a person, words alphabetically. An empty cell sorts last
 * either way, because a blank at the top of a sorted column reads as the
 * answer.
 */
export function sortTableRows(
  rows: readonly TableViewRow[],
  columns: readonly DisplayColumn[],
  sort: TableSort | null,
): TableViewRow[] {
  const column = sort ? columns.find((candidate) => candidate.key === sort.key) : undefined;
  if (!sort || !column) {
    return [...rows];
  }
  const direction = sort.direction === "asc" ? 1 : -1;
  const keyed = rows.map((row) => ({ row, key: sortKey(column.type, row.values[column.key]) }));
  keyed.sort((a, b) => {
    if (a.key === null || b.key === null) {
      return a.key === b.key ? 0 : a.key === null ? 1 : -1;
    }
    if (typeof a.key === "number" && typeof b.key === "number") {
      return (a.key - b.key) * direction;
    }

    return String(a.key).localeCompare(String(b.key)) * direction;
  });

  return keyed.map((entry) => entry.row);
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

export type DocumentArtifact = {
  body: string;
};

/** A write-up the agent published: markdown, read as it was written. */
export function documentFrom(artifact: AssistantArtifact): DocumentArtifact {
  return { body: stringOf(artifact.payload.body) };
}

/** One end of a shipment's route, and the time that matters there. */
export type RouteEnd = {
  city: string;
  place: string;
  /** Whether the time is when it happened or when it is due. */
  when: "departed" | "arrived" | "scheduled" | "";
  /** A local date and time, "2026-10-02T09:12". */
  at: string;
};

/** A fact on a record card, keyed so the reader names it in their own words. */
export type RecordFact = { key: string; value: string | number };

/**
 * A record laid out for its kind: a shipment's route and who is moving it, an
 * invoice's amount and whether it is paid, a billing item's amount and what
 * stands in its way. Null for a kind that has no view of its own.
 */
export type RecordView = {
  type: string;
  status: string;
  subtitle: string;
  from: RouteEnd | null;
  to: RouteEnd | null;
  progress: number | null;
  amount: { total: string; balance: string; currency: string } | null;
  ready: { canApprove: boolean; blockedBy: string; blockers: number } | null;
  facts: RecordFact[];
};

function routeEndOf(value: unknown): RouteEnd | null {
  if (!isRecord(value)) return null;
  const when = stringOf(value.when);
  return {
    city: stringOf(value.city),
    place: stringOf(value.place),
    when: when === "departed" || when === "arrived" || when === "scheduled" ? when : "",
    at: stringOf(value.at),
  };
}

function recordViewOf(value: unknown): RecordView | null {
  if (!isRecord(value) || stringOf(value.type) === "") return null;
  const amount = isRecord(value.amount) ? value.amount : null;
  const ready = isRecord(value.ready) ? value.ready : null;
  const facts = Array.isArray(value.facts)
    ? value.facts.flatMap((entry): RecordFact[] =>
        isRecord(entry) &&
        typeof entry.key === "string" &&
        (typeof entry.value === "string" || typeof entry.value === "number")
          ? [{ key: entry.key, value: entry.value }]
          : [],
      )
    : [];
  return {
    type: stringOf(value.type),
    status: stringOf(value.status),
    subtitle: stringOf(value.subtitle),
    from: routeEndOf(value.from),
    to: routeEndOf(value.to),
    progress: typeof value.progress === "number" ? Math.min(1, Math.max(0, value.progress)) : null,
    amount: amount
      ? {
          total: stringOf(amount.total),
          balance: stringOf(amount.balance),
          currency: stringOf(amount.currency),
        }
      : null,
    ready: ready
      ? {
          canApprove: ready.canApprove === true,
          blockedBy: stringOf(ready.blockedBy),
          blockers: typeof ready.blockers === "number" ? ready.blockers : 0,
        }
      : null,
    facts,
  };
}

export type EntityCardArtifact = {
  entity: string;
  /** The record laid out for its kind, when its kind has a layout. */
  view: RecordView | null;
  /** The record's readable fields, its name first; never its id. */
  fields: DisplayField[];
  /** Where the record opens in the app; empty when its kind has no page. */
  path: string;
  /** The record's id, for a body that reads the record live. */
  recordId: string;
};

/** A link the server put on an artifact, kept only when it stays in the app. */
function appPathOf(value: unknown): string {
  const path = stringOf(value);

  return isAppPath(path) ? path : "";
}

/**
 * A record as labelled values, which is how a person reads one.
 *
 * The server stores the fields it projected, so a new card is read as it is;
 * one stored before that holds the whole record and is projected here by the
 * same rules. The id is how the card opens, the tenancy and version are the
 * model's business, and nested structure is left out rather than shown as
 * JSON.
 */
export function entityCardFrom(artifact: AssistantArtifact): EntityCardArtifact {
  const payload = artifact.payload;
  const fields =
    readDisplayFields(payload.fields) ??
    projectRecord(isRecord(payload.record) ? payload.record : {});

  return {
    entity: stringOf(payload.entity),
    view: recordViewOf(payload.view),
    fields,
    path: appPathOf(payload.path),
    recordId: stringOf(payload.recordId),
  };
}

export type NavigationArtifact = {
  path: string;
  name: string;
  location: string;
};

/** Where the assistant took the person, or null when it names nowhere in the app. */
export function navigationFrom(artifact: AssistantArtifact): NavigationArtifact | null {
  const path = appPathOf(artifact.payload.path);
  if (path === "") {
    return null;
  }

  return {
    path,
    name: stringOf(artifact.payload.name) || artifact.title,
    location: stringOf(artifact.payload.location),
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
