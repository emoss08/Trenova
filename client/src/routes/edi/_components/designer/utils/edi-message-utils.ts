import type { EDIDiagnostic, EDIMessage, EDIX12EnvelopeSettings } from "@/types/edi";

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

export function parseX12Segments(
  rawX12: string,
  envelope?: Partial<EDIX12EnvelopeSettings> | null,
): ParsedX12Segment[] {
  const segmentTerminator = envelope?.segmentTerminator || "~";
  const elementSeparator = envelope?.elementSeparator || "*";

  return rawX12
    .split(segmentTerminator)
    .map((segment) => segment.trim())
    .filter(Boolean)
    .map((segment, index) => {
      const [segmentId = "", ...elements] = segment.split(elementSeparator);
      return {
        index: index + 1,
        segmentId,
        elements,
        raw: segment,
      };
    });
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

export function downloadText(filename: string, contents: string, type = "text/plain") {
  const blob = new Blob([contents], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export function downloadJson(filename: string, data: unknown) {
  downloadText(filename, JSON.stringify(data, null, 2), "application/json");
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
