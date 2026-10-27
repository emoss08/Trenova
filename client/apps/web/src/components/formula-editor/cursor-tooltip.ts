// cursor-tooltip.ts is a card opened from the keyboard at the cursor, where a pointer's hover
// card cannot reach: the hover card (`K` in Vim) and the line's problems (`gl`). It closes when
// the cursor moves, the text changes or Escape is pressed, as Neovim's floating windows do.

import { StateEffect, StateField, type Extension } from "@codemirror/state";
import { EditorView, keymap, showTooltip, type Tooltip } from "@codemirror/view";

const setCursorTooltip = StateEffect.define<Tooltip | null>();

const cursorTooltipField = StateField.define<Tooltip | null>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) {
      if (effect.is(setCursorTooltip)) return effect.value;
    }
    return transaction.docChanged || transaction.selection ? null : value;
  },
  provide: (field) => showTooltip.from(field),
});

/** Opens a card at `pos` (to `end`) that `create` draws. */
export function openCursorTooltip(
  view: EditorView,
  tooltip: { pos: number; end?: number; create: () => HTMLElement },
) {
  view.dispatch({
    effects: setCursorTooltip.of({
      pos: tooltip.pos,
      end: tooltip.end,
      above: true,
      create: () => ({ dom: tooltip.create() }),
    }),
  });
}

/** Whether a card opened from the keyboard is showing. */
export function hasCursorTooltip(view: EditorView): boolean {
  return view.state.field(cursorTooltipField, false) != null;
}

/** The card's state and Escape to close it; every feature that opens one includes it. */
export const cursorTooltip: Extension = [
  cursorTooltipField,
  keymap.of([
    {
      key: "Escape",
      run: (view) => {
        if (!hasCursorTooltip(view)) return false;
        view.dispatch({ effects: setCursorTooltip.of(null) });
        return true;
      },
    },
  ]),
];
