import { StreamLanguage } from "@codemirror/language";
import { Decoration, EditorView } from "@codemirror/view";
import type { EDIInspectionDiagnostic, EDIX12Inspection } from "@trenova/shared/types/edi";

export function x12StreamLanguage(delimiters: EDIX12Inspection["separators"]) {
  const delimiterChars = `${escapeRegex(delimiters.element)}${escapeRegex(
    delimiters.component,
  )}${escapeRegex(delimiters.repetition)}${escapeRegex(delimiters.segment)}`;
  const segmentIdPattern = new RegExp(`[A-Z0-9]{2,3}(?=[${delimiterChars}])`);
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
      if (stream.match(segmentIdPattern)) return "keyword";
      if (stream.match(separatorPattern)) return "separator";
      if (stream.match(terminatorPattern)) return "punctuation";
      if (stream.match(valuePattern)) return "string";
      stream.next();
      return null;
    },
  });
}

export function x12LineDecorations({
  inspection,
  selectedSegmentIndex,
  diagnostics,
}: {
  inspection: EDIX12Inspection;
  selectedSegmentIndex: number;
  diagnostics: EDIInspectionDiagnostic[];
}) {
  const diagnosticIndexes = new Set(diagnostics.map((diagnostic) => diagnostic.segmentIndex));
  const malformedIndexes = new Set(
    inspection.segments.filter((segment) => segment.malformed).map((segment) => segment.index),
  );
  const controlIndexes = new Set(
    inspection.segments
      .filter((segment) => ["interchange", "group", "transaction"].includes(segment.type))
      .map((segment) => segment.index),
  );

  return EditorView.decorations.compute([], (state) => {
    const decorations = [];
    for (const segment of inspection.segments) {
      const classes = ["cm-x12-line"];
      if (segment.index === selectedSegmentIndex) classes.push("cm-x12-selected");
      if (diagnosticIndexes.has(segment.index)) classes.push("cm-x12-diagnostic");
      if (malformedIndexes.has(segment.index)) classes.push("cm-x12-malformed");
      if (controlIndexes.has(segment.index)) classes.push("cm-x12-control");
      const from = Math.min(segment.startOffset, state.doc.length);
      const to = Math.min(Math.max(segment.endOffset, from), state.doc.length);
      decorations.push(Decoration.mark({ class: classes.join(" ") }).range(from, to));
    }
    return Decoration.set(decorations);
  });
}

export const x12ViewerTheme = EditorView.baseTheme({
  ".cm-x12-line": {
    borderBottom: "1px solid transparent",
  },
  ".cm-x12-selected": {
    backgroundColor: "var(--accent)",
    outline: "1px solid var(--primary)",
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
