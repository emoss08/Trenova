// tariff-highlight.ts colours the editor from the rules module's tokens (`tariff.tokens`). The
// module is asked once per text; only the tokens inside the visible ranges become decorations,
// so a long tariff costs what is on screen. Each token class takes the shared syntax role
// (`TARIFF_TOKEN_ROLES`), the same one `HighlightedCode` uses for read-only source.

import { RangeSetBuilder, type Extension } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  type DecorationSet,
  type ViewUpdate,
} from "@codemirror/view";
import { TARIFF_TOKEN_ROLES } from "@trenova/shared/lib/highlight/tariff";
import { tariffTokens, type TariffToken } from "@trenova/shared/rules/tariff";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";

const marks = new Map<string, Decoration>();

function markFor(token: TariffToken): Decoration | null {
  const role = TARIFF_TOKEN_ROLES[token.kind];
  if (!role && token.kind !== "doc" && token.kind !== "error") return null;
  const key = token.kind;
  let mark = marks.get(key);
  if (!mark) {
    const classes = [`cm-tariff-${token.kind}`, role ? `cm-syntax-${role}` : ""].filter(Boolean);
    mark = Decoration.mark({ class: classes.join(" ") });
    marks.set(key, mark);
  }
  return mark;
}

/** The decorations for the tokens that fall inside the visible ranges. */
export function visibleTokenDecorations(
  tokens: readonly TariffToken[],
  visible: readonly { from: number; to: number }[],
): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>();
  let index = 0;
  for (const range of visible) {
    while (index < tokens.length && (tokens[index]?.to ?? 0) <= range.from) index += 1;
    for (let at = index; at < tokens.length; at += 1) {
      const token = tokens[at];
      if (!token || token.from >= range.to) break;
      const mark = markFor(token);
      if (mark && token.to > token.from) builder.add(token.from, token.to, mark);
    }
  }
  return builder.finish();
}

/** Highlights a Tariff document from the module's tokens. */
export function tariffHighlight(getRules: () => RulesRuntime): Extension {
  const plugin = ViewPlugin.fromClass(
    class {
      tokens: TariffToken[];
      decorations: DecorationSet;

      constructor(view: EditorView) {
        this.tokens = tariffTokens(getRules(), view.state.doc.toString());
        this.decorations = visibleTokenDecorations(this.tokens, view.visibleRanges);
      }

      update(update: ViewUpdate) {
        if (update.docChanged) {
          this.tokens = tariffTokens(getRules(), update.state.doc.toString());
        }
        if (update.docChanged || update.viewportChanged) {
          this.decorations = visibleTokenDecorations(this.tokens, update.view.visibleRanges);
        }
      }
    },
    { decorations: (value) => value.decorations },
  );
  return [plugin, highlightTheme];
}

const highlightTheme = EditorView.baseTheme({
  ".cm-syntax-keyword": { color: "var(--syntax-keyword)" },
  ".cm-syntax-type": { color: "var(--syntax-type)" },
  ".cm-syntax-meta": { color: "var(--syntax-meta)" },
  ".cm-syntax-number": { color: "var(--syntax-number)" },
  ".cm-syntax-property": { color: "var(--syntax-property)" },
  ".cm-syntax-string": { color: "var(--syntax-string)" },
  ".cm-syntax-operator": { color: "var(--foreground-muted)" },
  ".cm-syntax-comment": { color: "var(--foreground-subtle)" },
  ".cm-tariff-doc": { fontStyle: "italic" },
  ".cm-tariff-code": { fontWeight: "500" },
});
