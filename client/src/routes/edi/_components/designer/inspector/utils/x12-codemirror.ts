import { StreamLanguage } from "@codemirror/language";
import { Decoration, EditorView } from "@codemirror/view";
import type { EDIDiagnostic } from "@/types/edi";
import type { ParsedX12Document, X12Delimiters } from "./x12-parser";

export function x12StreamLanguage(delimiters: X12Delimiters) {
  const separatorPattern = new RegExp(
    `[${escapeRegex(delimiters.element)}${escapeRegex(delimiters.component)}${escapeRegex(
      delimiters.repetition,
    )}]`,
  );
  const terminatorPattern = new RegExp(`[${escapeRegex(delimiters.segment)}]`);
  const valuePattern = new RegExp(
    `[^${escapeRegex(delimiters.element)}${escapeRegex(delimiters.component)}${escapeRegex(
      delimiters.repetition,
    )}${escapeRegex(delimiters.segment)}]+`,
  );

  return StreamLanguage.define({
    token(stream) {
      if (stream.sol() && stream.match(/[A-Z0-9]{2,3}/)) return "keyword";
      if (stream.match(separatorPattern)) return "separator";
      if (stream.match(terminatorPattern)) return "punctuation";
      if (stream.match(valuePattern)) return "string";
      stream.next();
      return null;
    },
  });
}

export function x12LineDecorations({
  document,
  selectedSegmentIndex,
  diagnostics,
}: {
  document: ParsedX12Document;
  selectedSegmentIndex: number;
  diagnostics: EDIDiagnostic[];
}) {
  const diagnosticSegmentIds = new Set(
    diagnostics.map((diagnostic) => diagnostic.segmentId).filter(Boolean),
  );
  const malformedIndexes = new Set(
    document.segments.filter((segment) => segment.malformed).map((segment) => segment.index),
  );
  const controlIndexes = new Set(
    document.segments.filter((segment) => segment.control).map((segment) => segment.index),
  );
  const diagnosticIndexes = new Set(
    document.segments
      .filter((segment) => diagnosticSegmentIds.has(segment.segmentId))
      .map((segment) => segment.index),
  );

  return EditorView.decorations.compute([], (state) => {
    const decorations = [];
    for (let lineNumber = 1; lineNumber <= state.doc.lines; lineNumber += 1) {
      const line = state.doc.line(lineNumber);
      const classes = ["cm-x12-line"];
      if (lineNumber === selectedSegmentIndex) classes.push("cm-x12-selected");
      if (diagnosticIndexes.has(lineNumber)) classes.push("cm-x12-diagnostic");
      if (malformedIndexes.has(lineNumber)) classes.push("cm-x12-malformed");
      if (controlIndexes.has(lineNumber)) classes.push("cm-x12-control");
      decorations.push(Decoration.line({ class: classes.join(" ") }).range(line.from));
    }
    return Decoration.set(decorations);
  });
}

export const x12ViewerTheme = EditorView.baseTheme({
  ".cm-x12-line": {
    borderLeft: "3px solid transparent",
  },
  ".cm-x12-selected": {
    backgroundColor: "var(--accent)",
    borderLeftColor: "var(--primary)",
  },
  ".cm-x12-diagnostic": {
    backgroundColor: "color-mix(in oklab, var(--warning) 14%, transparent)",
  },
  ".cm-x12-malformed": {
    backgroundColor: "color-mix(in oklab, var(--destructive) 14%, transparent)",
  },
  ".cm-x12-control": {
    color: "var(--primary)",
  },
  ".ͼb": {
    color: "var(--primary)",
    fontWeight: "600",
  },
  ".ͼd": {
    color: "var(--muted-foreground)",
  },
});

function escapeRegex(value: string) {
  return value.replace(/[|\\{}()[\]^$+*?.-]/g, "\\$&");
}
