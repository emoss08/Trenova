import { generateDateOnly, getEndOfDay, getStartOfDay } from "@trenova/shared/lib/date";
import type { EDIDiagnostic, EDIMessage, EDIX12EnvelopeSettings } from "@trenova/shared/types/edi";
import { parseX12Document } from "../inspector/utils/x12-parser";

export type ParsedX12Segment = {
  index: number;
  segmentId: string;
  elements: string[];
  raw: string;
};

export type DiagnosticGroup = {
  key: string;
  severity: EDIDiagnostic["severity"];
  segmentId: string;
  elementPosition: number;
  code: string;
  path: string;
  diagnostics: EDIDiagnostic[];
};

export type ArchiveMessagesQueryFilters = {
  partnerId?: string;
  transactionSet?: string;
  direction?: string;
  status?: string;
  generatedFrom?: string;
  generatedTo?: string;
  query?: string;
  limit?: number;
};

export function formatRawX12Display(
  rawX12: string,
  envelope?: Partial<EDIX12EnvelopeSettings> | null,
) {
  return parseX12Document(rawX12, envelope)
    .segments.map((segment) => segment.rawWithTerminator)
    .join("\n");
}

export function parseX12Segments(
  rawX12: string,
  envelope?: Partial<EDIX12EnvelopeSettings> | null,
): ParsedX12Segment[] {
  return parseX12Document(rawX12, envelope).segments.map((segment) => ({
    index: segment.index,
    segmentId: segment.segmentId,
    elements: segment.elements.map((element) => element.value),
    raw: segment.raw,
  }));
}

export function groupDiagnostics(diagnostics: EDIDiagnostic[]): DiagnosticGroup[] {
  const groups = new Map<string, DiagnosticGroup>();
  for (const diagnostic of diagnostics) {
    const segmentId = diagnostic.segmentId ?? "";
    const elementPosition = diagnostic.elementPosition ?? 0;
    const path = diagnostic.path ?? "";
    const key = [diagnostic.severity, segmentId, elementPosition, diagnostic.code, path].join(":");
    const group = groups.get(key);
    if (group) {
      group.diagnostics.push(diagnostic);
      continue;
    }
    groups.set(key, {
      key,
      severity: diagnostic.severity,
      segmentId,
      elementPosition,
      code: diagnostic.code,
      path,
      diagnostics: [diagnostic],
    });
  }
  return Array.from(groups.values()).sort((a, b) => {
    const severityOrder = severityRank(a.severity) - severityRank(b.severity);
    if (severityOrder !== 0) return severityOrder;
    return a.key.localeCompare(b.key);
  });
}

export function buildX12Filename(
  message: Pick<EDIMessage, "transactionSet" | "transactionControlNumber" | "id">,
) {
  const controlNumber = message.transactionControlNumber || message.id;
  return `edi-${message.transactionSet}-${controlNumber}.x12`;
}

export function buildMessageJsonFilename(message: Pick<EDIMessage, "id">) {
  return `edi-message-${message.id}.json`;
}

export function buildArchiveMessagesQueryString(filters: ArchiveMessagesQueryFilters) {
  const params = new URLSearchParams({ limit: String(filters.limit ?? 50) });
  if (filters.transactionSet) params.set("transactionSet", filters.transactionSet);
  if (filters.direction) params.set("direction", filters.direction);
  if (filters.partnerId) params.set("partnerId", filters.partnerId);
  if (filters.status) params.set("status", filters.status);
  if (filters.query?.trim()) params.set("query", filters.query.trim());

  const generatedFrom = parseDateBoundary(filters.generatedFrom, "start");
  const generatedTo = parseDateBoundary(filters.generatedTo, "end");
  if (generatedFrom) params.set("generatedFrom", String(generatedFrom));
  if (generatedTo) params.set("generatedTo", String(generatedTo));

  return `?${params.toString()}`;
}

function severityRank(severity: EDIDiagnostic["severity"]) {
  switch (severity) {
    case "Error":
      return 0;
    case "Warning":
      return 1;
    case "Info":
      return 2;
  }
}

function parseDateBoundary(value: string | undefined, boundary: "start" | "end") {
  if (!value?.trim()) return 0;
  const date = generateDateOnly(value);
  if (!date) return 0;
  return boundary === "start" ? getStartOfDay(date) : getEndOfDay(date);
}
