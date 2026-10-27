// lading-highlight.ts colours the editor from the EDI module's tokens (`lading.tokens`), asked
// once per text; only the tokens inside the visible ranges become decorations. Each kind takes
// the shared syntax role, so Lading reads in the same colours as every other source.

import { RangeSetBuilder, type Extension } from "@codemirror/state";
import {
  Decoration,
  ViewPlugin,
  type DecorationSet,
  type EditorView,
  type ViewUpdate,
} from "@codemirror/view";
import { ladingTokens, type LadingToken } from "@trenova/shared/edi/lading";
import type { EdiRuntime } from "@trenova/shared/edi/runtime";
import type { CodeRole, CodeSpan } from "@trenova/shared/lib/highlight/code";
import { syntaxHighlightTheme } from "../formula-editor/tariff-highlight";
import { unlessRefused } from "./lading-service";

/** The syntax role each token kind is drawn in. */
export const LADING_TOKEN_ROLES: Record<string, CodeRole> = {
  keyword: "keyword",
  type: "type",
  x12: "property",
  loop: "meta",
  number: "number",
  date: "number",
  text: "string",
  operator: "operator",
  comment: "comment",
  doc: "comment",
};

const marks = new Map<string, Decoration>();

function markFor(kind: string): Decoration | null {
  const role = LADING_TOKEN_ROLES[kind];
  if (!role) return null;
  let mark = marks.get(kind);
  if (!mark) {
    mark = Decoration.mark({ class: `cm-lading-${kind} cm-syntax-${role}` });
    marks.set(kind, mark);
  }
  return mark;
}

/** The module's tokens as spans `HighlightedCode` draws. */
export function ladingCodeSpans(tokens: readonly LadingToken[]): CodeSpan[] {
  return tokens.map((token) => ({
    from: token.range.from,
    to: token.range.to,
    role: LADING_TOKEN_ROLES[token.kind],
  }));
}

/** The decorations for the tokens that fall inside the visible ranges. */
export function visibleLadingDecorations(
  tokens: readonly LadingToken[],
  visible: readonly { from: number; to: number }[],
): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>();
  let index = 0;
  for (const range of visible) {
    while (index < tokens.length && (tokens[index]?.range.to ?? 0) <= range.from) index += 1;
    for (let at = index; at < tokens.length; at += 1) {
      const token = tokens[at];
      if (!token || token.range.from >= range.to) break;
      const mark = markFor(token.kind);
      if (mark && token.range.to > token.range.from) {
        builder.add(token.range.from, token.range.to, mark);
      }
    }
  }
  return builder.finish();
}

function tokensOf(edi: EdiRuntime, source: string): LadingToken[] {
  return unlessRefused(() => ladingTokens(edi, source), []);
}

/** Highlights a Lading file from the module's tokens. */
export function ladingHighlight(getEdi: () => EdiRuntime): Extension {
  const plugin = ViewPlugin.fromClass(
    class {
      tokens: LadingToken[];
      decorations: DecorationSet;

      constructor(view: EditorView) {
        this.tokens = tokensOf(getEdi(), view.state.doc.toString());
        this.decorations = visibleLadingDecorations(this.tokens, view.visibleRanges);
      }

      update(update: ViewUpdate) {
        if (update.docChanged) {
          this.tokens = tokensOf(getEdi(), update.state.doc.toString());
        }
        if (update.docChanged || update.viewportChanged) {
          this.decorations = visibleLadingDecorations(this.tokens, update.view.visibleRanges);
        }
      }
    },
    { decorations: (value) => value.decorations },
  );
  return [plugin, syntaxHighlightTheme];
}
