import type { ReportPreviewColumn } from "@/lib/graphql/reports";
import type { AssistantArtifact } from "@/types/assistant";

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
