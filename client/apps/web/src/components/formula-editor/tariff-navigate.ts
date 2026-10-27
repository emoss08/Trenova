// tariff-navigate.ts is go to definition (F12, or Ctrl/Cmd-click) from `tariff.definition`,
// which lands in this file or opens the imported library the name is declared in, and rename
// (F2) from `tariff.rename`, which edits every reference the checker resolved at once or says
// why the name cannot be renamed.

import { StateEffect, StateField, type Extension } from "@codemirror/state";
import { EditorView, keymap, showPanel, type Command, type Panel } from "@codemirror/view";
import { translateMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import { tariffDefinition, tariffPrepareRename, tariffRename } from "@trenova/shared/rules/tariff";
import { unlessUnanswered } from "./tariff-doc";
import { showEditorNotice } from "./tariff-notice";
import { tariffServiceFacet, type TariffLanguageService } from "./tariff-service";

/** Moves the cursor onto a range, scrolls it into view and focuses the editor. */
export function revealRange(view: EditorView, range: { from: number; to: number }) {
  const length = view.state.doc.length;
  const from = Math.max(0, Math.min(range.from, length));
  const to = Math.max(from, Math.min(range.to, length));
  view.dispatch({
    selection: { anchor: from, head: to },
    effects: EditorView.scrollIntoView(from, { y: "center" }),
  });
  view.focus();
}

/** Goes to where the name at `position` is declared; false when it has no declaration. */
export function goToDefinition(
  service: TariffLanguageService,
  view: EditorView,
  position: number,
): boolean {
  const targets = unlessUnanswered(
    () =>
      tariffDefinition(service.rules(), {
        source: view.state.doc.toString(),
        offset: position,
        context: service.context(),
        kind: service.kind(),
      }),
    [],
  );
  const target = targets[0];
  if (!target) return false;
  if (target.module) {
    const word = view.state.wordAt(position);
    service.openDefinition({
      module: target.module,
      range: target.selectionRange,
      name: word ? view.state.sliceDoc(word.from, word.to) : "",
    });
  } else {
    revealRange(view, target.selectionRange);
  }
  return true;
}

/**
 * Goes to the declaration of the name under the cursor (F12, `gd` in Vim, and the editor's menu),
 * or says there is none.
 */
export const goToDefinitionAtCursor: Command = (view) => {
  const service = view.state.facet(tariffServiceFacet);
  if (!service) return false;
  if (goToDefinition(service, view, view.state.selection.main.head)) return true;
  showEditorNotice(view, translate("There is no declaration to go to here"));
  return true;
};

type RenameState = { offset: number; name: string; refusal: string | null } | null;

const openRename = StateEffect.define<RenameState>();

const renameField = StateField.define<RenameState>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) if (effect.is(openRename)) return effect.value;
    return transaction.docChanged ? null : value;
  },
  provide: (field) =>
    showPanel.from(field, (state) => (state ? (view) => renamePanel(view, state) : null)),
});

function renamePanel(view: EditorView, state: NonNullable<RenameState>): Panel {
  const dom = document.createElement("form");
  dom.className = "cm-tariff-rename";
  const label = document.createElement("label");
  label.textContent = translate("Rename {0} to", state.name);
  const input = document.createElement("input");
  input.className = "cm-tariff-rename-input";
  input.value = state.name;
  input.setAttribute("aria-label", translate("New name"));
  label.appendChild(input);
  dom.appendChild(label);
  if (state.refusal) {
    const refusal = document.createElement("p");
    refusal.className = "cm-tariff-rename-refusal";
    refusal.setAttribute("role", "alert");
    refusal.textContent = state.refusal;
    dom.appendChild(refusal);
  }
  const close = () => {
    view.dispatch({ effects: openRename.of(null) });
    view.focus();
  };
  input.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      close();
    }
  });
  dom.addEventListener("submit", (event) => {
    event.preventDefault();
    const service = view.state.facet(tariffServiceFacet);
    const name = input.value.trim();
    if (!service || name === "" || name === state.name) {
      close();
      return;
    }
    const answer = unlessUnanswered(
      () =>
        tariffRename(service.rules(), {
          source: view.state.doc.toString(),
          offset: state.offset,
          newName: name,
          context: service.context(),
          kind: service.kind(),
        }),
      null,
    );
    if (!answer) {
      close();
      return;
    }
    if ("error" in answer) {
      view.dispatch({
        effects: openRename.of({ ...state, refusal: translateMessage(answer.error) }),
      });
      return;
    }
    view.dispatch({
      changes: answer.edits.map((change) => ({
        from: change.range.from,
        to: change.range.to,
        insert: change.insert,
      })),
      userEvent: "rename",
      effects: openRename.of(null),
    });
    view.focus();
  });
  return {
    dom,
    top: true,
    mount: () => {
      input.focus();
      input.select();
    },
  };
}

/**
 * Opens rename for the name under the cursor (or at `offset`), after the module says it can be
 * renamed; a name that cannot (a schema root, a standard library name, a keyword) opens the panel
 * with why.
 */
export function startRename(
  view: EditorView,
  offset: number = view.state.selection.main.head,
): boolean {
  const service = view.state.facet(tariffServiceFacet);
  if (!service) return false;
  const prepared = unlessUnanswered(
    () =>
      tariffPrepareRename(service.rules(), {
        source: view.state.doc.toString(),
        offset,
        context: service.context(),
        kind: service.kind(),
      }),
    null,
  );
  if (!prepared) return false;
  view.dispatch({
    effects: openRename.of(
      "error" in prepared
        ? { offset, name: "", refusal: translateMessage(prepared.error) }
        : { offset, name: prepared.placeholder, refusal: null },
    ),
  });
  return true;
}

export function tariffNavigation(service: TariffLanguageService): Extension {
  return [
    tariffServiceFacet.of(service),
    renameField,
    keymap.of([
      { key: "F12", run: goToDefinitionAtCursor },
      { key: "F2", run: startRename },
    ]),
    EditorView.domEventHandlers({
      mousedown: (event, view) => {
        if (!(event.metaKey || event.ctrlKey) || event.button !== 0) return false;
        const position = view.posAtCoords({ x: event.clientX, y: event.clientY });
        if (position === null) return false;
        if (!goToDefinition(service, view, position)) return false;
        event.preventDefault();
        return true;
      },
    }),
  ];
}
