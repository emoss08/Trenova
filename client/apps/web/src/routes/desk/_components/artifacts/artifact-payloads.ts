import { readStoredTariffBill, type StoredTariffBill } from "@trenova/shared/types/tariff-form";
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
import type { AnalysisView } from "@/components/assistant/analysis-result";
import { isRecordEntityType, recordPath, type RecordEntityType } from "@/config/record-links";
import {
  analysisChartsSchema,
  analysisDatasetsSchema,
  analysisFiguresSchema,
  analysisLayoutSchema,
  analysisRunStatusSchema,
  analysisTablesSchema,
} from "@/types/analysis";
import { isAppPath } from "@/lib/app-path";
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

/** A column of a report_preview payload: its value type, and its resolved display style as `format`. */
export type ReportPreviewArtifactColumn = {
  id: string;
  label: string;
  type: string;
  format: string | null;
};

export type ReportPreviewArtifact = {
  name: string;
  dataset: string;
  columns: ReportPreviewArtifactColumn[];
  rows: unknown[][];
  totals: unknown[] | null;
  rowCount: number;
  truncated: boolean;
};

function previewColumn(value: unknown): ReportPreviewArtifactColumn | null {
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
  };
}

/** Rows arrive keyed by column label; the grid wants them positional. */
export function reportPreviewFrom(artifact: AssistantArtifact): ReportPreviewArtifact {
  const payload = artifact.payload;
  const columns = listOf(payload.columns)
    .map(previewColumn)
    .filter((column): column is ReportPreviewArtifactColumn => column !== null);
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
  /** The record's id, for a table whose rows are acted on in place. */
  recordId?: string;
};

/**
 * Where a table's list goes on from the rows it holds. The agent's pages are
 * kept with the next cursor or offset after them; a table cut to fit has rows
 * past what it holds and no place to go on from, so the person's pages read
 * the list from its start.
 */
export type TablePaging = {
  hasMore: boolean;
  nextCursor: string;
  nextOffset: number | null;
  /** How many rows the list holds in all, when the agent's read counted them. */
  total: number | null;
  /** True when total is the server's estimate of a list too large to count. */
  totalEstimated: boolean;
  /** How many pages of the list the agent read into the table. */
  pages: number;
};

/** The agent that read a table's rows in the conversation, named as it was when it read them. */
export type ArtifactReader = { id: string; name: string; icon: string; accent: string };

export type TableViewArtifact = {
  tool: string;
  entity: string;
  /** The record kind the rows are, when the registry knows it. */
  recordEntity: string;
  searchedFor: string[];
  columns: DisplayColumn[];
  rows: TableViewRow[];
  rowCount: number;
  truncated: boolean;
  /** How many tool calls the server read this table together from; zero for one search's result. */
  calls: number;
  paging: TablePaging;
  /** Null when the table does not say, or the agent could no longer be read when it was kept. */
  readBy: ArtifactReader | null;
  /** Whether the list can be read again: the table kept the arguments it was read with. */
  rereadable: boolean;
};

/** Whether the person's pages go on from the agent's rows, or read the list from its start. */
export function resumesAfterAgent(paging: TablePaging): boolean {
  return paging.nextCursor !== "" || paging.nextOffset !== null;
}

export function artifactReaderOf(value: unknown): ArtifactReader | null {
  if (!isRecord(value)) {
    return null;
  }
  const name = stringOf(value.name).trim();
  if (name === "") {
    return null;
  }

  return {
    id: stringOf(value.id),
    name,
    icon: stringOf(value.icon),
    accent: stringOf(value.accent),
  };
}

function tablePagingOf(payload: Record<string, unknown>): TablePaging {
  const nextOffset = payload.nextOffset;
  const total = payload.total;

  return {
    hasMore: payload.hasMore === true,
    nextCursor: stringOf(payload.nextCursor),
    nextOffset: typeof nextOffset === "number" && Number.isInteger(nextOffset) ? nextOffset : null,
    total: typeof total === "number" && Number.isInteger(total) ? total : null,
    totalEstimated: payload.totalEstimated === true,
    pages: numberOf(payload.pages) || 1,
  };
}

/**
 * Rows of a list as the table reads them: each value projected against its
 * column, and the record's page built from the registry. Keys start at
 * `from`, so rows read on later pages never take a key the agent's rows hold.
 */
export function tableRowsFrom(
  raw: readonly Record<string, unknown>[],
  columns: readonly DisplayColumn[],
  entity: RecordEntityType | null,
  from = 0,
): TableViewRow[] {
  return raw.map((row, index) => {
    const values: Record<string, unknown> = {};
    for (const column of columns) {
      const value = projectValue(column.type, row[column.key]);
      if (value !== undefined) {
        values[column.key] = value;
      }
    }
    const id = stringOf(row[RECORD_ID_KEY]);

    return {
      key: String(from + index),
      values,
      path: entity !== null && id !== "" ? recordPath(entity, id) : "",
      recordId: id,
    };
  });
}

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
export function tableRecordEntity(payload: Record<string, unknown>): RecordEntityType | null {
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
  const rows = tableRowsFrom(raw, columns, entity);
  const rowCount = numberOf(payload.rowCount) || rows.length;

  return {
    tool: stringOf(payload.tool),
    entity: stringOf(payload.entity),
    recordEntity: entity ?? "",
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
    paging: tablePagingOf(payload),
    readBy: artifactReaderOf(payload.readBy),
    rereadable: isRecord(payload.readWith),
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
  /** How many records the view holds, when the view was counted; null for one stored before views were. */
  count: number | null;
  /** The count stopped at its limit, so there are more than it says. */
  countCapped: boolean;
  /**
   * The columns a preview row shows, in the card's three places: the first
   * column as the row's name, one to read beside it, and its status at the
   * end when it has one. Empty when the view was not run.
   */
  columns: DisplayColumn[];
  /** The first records it holds, to read before opening it. */
  preview: TableViewRow[];
};

/**
 * The columns a view's preview row has room for. The card draws a row as a
 * name, a line beside it and a mark at the end, so a table of many columns is
 * read down to the three that say most: its first column, its status when it
 * has one, and the next column between them.
 */
function previewColumns(columns: readonly DisplayColumn[]): DisplayColumn[] {
  const [first, ...rest] = columns;
  if (!first) {
    return [];
  }
  const end = rest.find((column) => column.type === "status") ?? rest[1];
  const middle = rest.find((column) => column !== end);

  return [first, middle, end].filter((column): column is DisplayColumn => column !== undefined);
}

export function composedViewFrom(artifact: AssistantArtifact): ComposedViewArtifact | null {
  const path = stringOf(artifact.payload.path);
  if (path === "") {
    return null;
  }
  const payload = artifact.payload;
  const table = listOf(payload.rows).length > 0 ? tableViewFrom(artifact) : null;

  return {
    entity: stringOf(payload.entity),
    path,
    explanation: stringOf(payload.explanation),
    terms: listOf(payload.terms).filter(
      (term): term is string => typeof term === "string" && term !== "",
    ),
    filterCount: numberOf(payload.filterCount),
    unresolved: listOf(payload.unresolved)
      .filter(isRecord)
      .map((entry) => ({ phrase: stringOf(entry.phrase), reason: stringOf(entry.reason) }))
      .filter((entry) => entry.phrase !== ""),
    count:
      typeof payload.rowCount === "number" && Number.isFinite(payload.rowCount)
        ? payload.rowCount
        : null,
    countCapped: payload.countCapped === true,
    columns: table ? previewColumns(table.columns) : [],
    preview: table?.rows ?? [],
  };
}

export type RateExplanationArtifact = {
  shipmentId: string;
  side: string;
  currency: string;
  winner: { agreementCode: string; agreementName: string; ruleLabel: string } | null;
  tieBreak: string;
  rejected: { agreementCode: string; ruleLabel: string; reason: string; detail: string }[];
  /** The winning tariff's bill: its charges, notes and adjustments, in wire form; null when unread. */
  bill: StoredTariffBill | null;
  totals: { linehaul: string; fuel: string; accessorial: string; total: string };
  warnings: string[];
  /** How a shipment no agreement priced got its price: its tariff, or an entered amount. */
  pricedBy: {
    method: string;
    explanation: string;
    override: string;
    overrideReason: string;
  } | null;
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
  const pricedBy = isRecord(payload.pricedBy) ? payload.pricedBy : null;

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
    bill: readStoredTariffBill(payload.bill),
    totals: totalsOf(payload.totals),
    warnings: listOf(payload.warnings).filter(
      (warning): warning is string => typeof warning === "string" && warning !== "",
    ),
    pricedBy:
      pricedBy === null
        ? null
        : {
            method: stringOf(pricedBy.method),
            explanation: stringOf(pricedBy.explanation),
            override: amountOf(pricedBy.override),
            overrideReason: stringOf(pricedBy.overrideReason),
          },
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

/** The subject and body as a draft shows them, or as a person changed them. */
export type EmailWording = { subject: string; body: string };

/**
 * The wording a draft opens with: what the person saved over the agent's, kept
 * on the proposal behind the draft, or the agent's own where they changed
 * nothing.
 */
export function savedEmailWording(
  draft: EmailDraftArtifact,
  saved: Record<string, unknown> | null | undefined,
): EmailWording {
  return {
    subject: typeof saved?.subject === "string" ? saved.subject : draft.subject,
    body: typeof saved?.body === "string" ? saved.body : draft.body,
  };
}

/**
 * What a person changed on a draft, keyed by the proposal's parameter: only
 * what differs from the agent's wording, so putting it back clears the edit.
 * A part the proposal does not carry, such as a body written from the
 * organization's template, is never sent.
 */
export function emailDraftEdits(
  draft: EmailDraftArtifact,
  wording: EmailWording,
  editable: { subject: boolean; body: boolean },
): Record<string, unknown> {
  const edits: Record<string, unknown> = {};
  if (editable.subject && wording.subject !== draft.subject) edits.subject = wording.subject;
  if (editable.body && wording.body !== draft.body) edits.body = wording.body;
  return edits;
}

export type DocumentArtifact = {
  body: string;
  /** What kind of write-up it is: "Brief", "Handover". */
  docType: string;
  /** Who wrote it, and what from. */
  author: string;
  basis: string;
  /** Who wrote this version: the agent, or a person editing on the Desk. */
  editedBy: "agent" | "person";
  editor: string;
  versionNote: string;
  sources: DocumentSource[];
};

/** What a citation mark in a document points to. */
export type DocumentSource = {
  n: number;
  tool: string;
  label: string;
  detail: string;
  artifactId: string;
};

/** A write-up the agent published: markdown, read as it was written. */
export function documentFrom(artifact: AssistantArtifact): DocumentArtifact {
  const payload = artifact.payload;
  const sources = Array.isArray(payload.sources) ? payload.sources : [];
  return {
    body: stringOf(payload.body),
    docType: stringOf(payload.docType),
    author: stringOf(payload.author),
    basis: stringOf(payload.basis),
    editedBy: payload.editedBy === "person" ? "person" : "agent",
    editor: stringOf(payload.editor),
    versionNote: stringOf(payload.versionNote),
    sources: sources
      .filter(
        (source): source is Record<string, unknown> =>
          typeof source === "object" && source !== null,
      )
      .map((source) => ({
        n: numberOf(source.n),
        tool: stringOf(source.tool),
        label: stringOf(source.label),
        detail: stringOf(source.detail),
        artifactId: stringOf(source.artifactId),
      }))
      .filter((source) => source.n > 0 && source.label !== ""),
  };
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

/** A stop a shipment makes between its pickup and its delivery. */
export type RouteStop = RouteEnd & { type: string; done: boolean };

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
  via: RouteStop[];
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
    via: Array.isArray(value.via)
      ? value.via.flatMap((entry): RouteStop[] => {
          const end = routeEndOf(entry);
          return end && isRecord(entry) && (end.city !== "" || end.place !== "")
            ? [{ ...end, type: stringOf(entry.type), done: entry.done === true }]
            : [];
        })
      : [],
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
    .map((change) => {
      const keyValues = listOf(change.key).map(stringOf);
      return {
        kind: stringOf(change.kind),
        key: keyValues.join(" · "),
        keyValues,
        measures: diffMeasures(change.measures),
      };
    });

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

/**
 * What a script showed: its headline figures, tables and charts as what they plot, the
 * exception it raised, and the code. A status the server did not send reads as a failure,
 * never as a success.
 */
export function analysisFrom(artifact: AssistantArtifact): AnalysisView {
  const payload = artifact.payload;
  const status = analysisRunStatusSchema.catch("Failed").parse(payload.status);
  const error = isRecord(payload.error) ? payload.error : null;

  return {
    runId: stringOf(payload.analysisRunId),
    status,
    title: stringOf(payload.title) || artifact.title,
    code: stringOf(payload.code),
    datasets: analysisDatasetsSchema.catch([]).parse(payload.datasets),
    output: stringOf(payload.output),
    tables: analysisTablesSchema.catch([]).parse(payload.tables),
    figures: analysisFiguresSchema.catch([]).parse(payload.figures),
    charts: analysisChartsSchema.catch([]).parse(payload.charts),
    layout: analysisLayoutSchema.catch([]).parse(payload.layout),
    error: error
      ? {
          name: stringOf(error.name),
          value: stringOf(error.value),
          traceback: stringOf(error.traceback),
        }
      : null,
    failure: stringOf(payload.failure),
    truncated: payload.truncated === true,
  };
}

/** A portal task as the agent was told of it: the card reads the run itself for how it stands. */
export type PortalTaskPayload = {
  runId: string;
  mode: "Read" | "Act";
  site: string;
  question: string;
  answer: string;
  pagesRead: string[];
  awaitingSignInCode: boolean;
  portalSaid: string;
  retrievedOn: string;
};

/**
 * What a portal task carried when the agent saw it: a read's question and the portal's answer,
 * or that it stopped on a sign-in code; an approved action only its site. Anything else reads
 * as absent.
 */
export function portalTaskFrom(artifact: AssistantArtifact): PortalTaskPayload {
  const payload = artifact.payload;

  return {
    runId: stringOf(payload.portalRunId),
    mode: payload.mode === "Act" ? "Act" : "Read",
    site: stringOf(payload.site),
    question: stringOf(payload.question),
    answer: stringOf(payload.answer),
    pagesRead: listOf(payload.pagesRead)
      .map(stringOf)
      .filter((url) => url !== ""),
    awaitingSignInCode: payload.awaitingSignInCode === true,
    portalSaid: stringOf(payload.portalSaid),
    retrievedOn: stringOf(payload.retrievedOn),
  };
}
