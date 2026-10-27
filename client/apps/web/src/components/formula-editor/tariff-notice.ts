// tariff-notice.ts is the editor's one-line answer when a command has nothing to do at the cursor
// ("Nothing here can be promoted to an input"): a small tooltip at the cursor that leaves with
// the next edit, cursor move or click, or after a few seconds, so it never stays in the way.

import { StateEffect, StateField, type Extension } from "@codemirror/state";
import { EditorView, showTooltip, type Tooltip } from "@codemirror/view";

type Notice = { pos: number; text: string } | null;

const setNotice = StateEffect.define<Notice>();

/** How long a notice stays when nothing else clears it. */
export const NOTICE_MS = 2500;

const noticeField = StateField.define<Notice>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) if (effect.is(setNotice)) return effect.value;
    return transaction.docChanged || transaction.selection ? null : value;
  },
  provide: (field) =>
    showTooltip.from(field, (notice): Tooltip | null =>
      notice
        ? {
            pos: notice.pos,
            above: true,
            create: () => {
              const dom = document.createElement("div");
              dom.className = "cm-tariff-notice";
              dom.setAttribute("role", "status");
              dom.textContent = notice.text;
              return { dom };
            },
          }
        : null,
    ),
});

/** Says, at the cursor, why a command did nothing; it goes on its own. */
export function showEditorNotice(view: EditorView, text: string): void {
  if (view.state.field(noticeField, false) === undefined) return;
  const pos = view.state.selection.main.head;
  view.dispatch({ effects: setNotice.of({ pos, text }) });
  window.setTimeout(() => {
    if (!view.dom.isConnected) return;
    const current = view.state.field(noticeField, false);
    if (current && current.pos === pos && current.text === text) {
      view.dispatch({ effects: setNotice.of(null) });
    }
  }, NOTICE_MS);
}

/** The notice's state and its dismissal on a click; every editor that shows notices has it. */
export function tariffNotices(): Extension {
  return [
    noticeField,
    EditorView.domEventHandlers({
      mousedown: (_event, view) => {
        if (view.state.field(noticeField, false)) view.dispatch({ effects: setNotice.of(null) });
        return false;
      },
    }),
  ];
}
