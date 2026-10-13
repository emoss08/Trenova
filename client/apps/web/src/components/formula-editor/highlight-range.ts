import { StateEffect, StateField, type Extension } from "@codemirror/state";
import { Decoration, EditorView, type DecorationSet } from "@codemirror/view";

export type EditorRange = { from: number; to: number };

/** Marks one range of the expression, or clears the mark with null. */
export const setHighlightRange = StateEffect.define<EditorRange | null>();

const highlightMark = Decoration.mark({ class: "cm-focus-range" });

const highlightField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(decorations, transaction) {
    let next = decorations.map(transaction.changes);
    for (const effect of transaction.effects) {
      if (!effect.is(setHighlightRange)) continue;
      const range = effect.value;
      const length = transaction.state.doc.length;
      if (range === null || range.from >= range.to || range.from >= length) {
        next = Decoration.none;
      } else {
        next = Decoration.set([highlightMark.range(range.from, Math.min(range.to, length))]);
      }
    }
    return next;
  },
  provide: (field) => EditorView.decorations.from(field),
});

const highlightTheme = EditorView.baseTheme({
  ".cm-focus-range": {
    backgroundColor: "var(--surface-selected)",
    borderRadius: "2px",
    boxShadow: "0 0 0 1px var(--border-strong)",
  },
});

/**
 * Lights one statement of the expression while something outside the editor
 * points at it, such as a row of Nova's explanation under the pointer.
 */
export function highlightRangeExtension(): Extension {
  return [highlightField, highlightTheme];
}
