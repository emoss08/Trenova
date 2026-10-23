// tariff-actions.ts is Promote to input (Mod-., or the editor's menu): it asks `tariff.actions`
// what the selection, or the literal at the cursor, can become, offers the refactorings in a small
// menu at the cursor, applies the chosen one's edits as one undoable change, and then opens rename
// on the new input's name so the person names it at once. With nothing to offer it says so at the
// cursor and goes away.

import { Prec, StateEffect, StateField, type Extension } from "@codemirror/state";
import { EditorView, keymap, showTooltip, type Tooltip } from "@codemirror/view";
import { translate } from "@trenova/shared/i18n/runtime";
import { tariffActions, type TariffAction } from "@trenova/shared/rules/tariff";
import { appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import { startRename } from "./tariff-navigate";
import { showEditorNotice } from "./tariff-notice";
import { tariffServiceFacet } from "./tariff-service";

/** The key that asks for the refactorings at the cursor. */
export const PROMOTE_KEY = "Mod-.";

type ActionMenu = { pos: number; actions: readonly TariffAction[]; active: number } | null;

const setActionMenu = StateEffect.define<ActionMenu>();

const actionMenuField = StateField.define<ActionMenu>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) if (effect.is(setActionMenu)) return effect.value;
    return transaction.docChanged || transaction.selection ? null : value;
  },
  provide: (field) =>
    showTooltip.from(field, (menu): Tooltip | null =>
      menu ? { pos: menu.pos, above: false, create: (view) => renderMenu(view, menu) } : null,
    ),
});

/** The refactorings the module offers for the selection, or for the literal at the cursor. */
export function actionsAtSelection(view: EditorView): TariffAction[] {
  const service = view.state.facet(tariffServiceFacet);
  if (!service) return [];
  const { from, to } = view.state.selection.main;
  return unlessUnanswered(
    () =>
      tariffActions(service.rules(), {
        source: view.state.doc.toString(),
        from,
        to,
        context: service.context(),
        kind: service.kind(),
      }),
    [],
  );
}

/**
 * Applies one refactoring's edits as one change in the undo history, selects what it asks to
 * select (the new input's name) and opens rename there.
 */
export function applyTariffAction(view: EditorView, action: TariffAction): void {
  const length = view.state.doc.length;
  view.dispatch({
    changes: action.edits.map((edit) => ({
      from: Math.min(edit.range.from, length),
      to: Math.min(edit.range.to, length),
      insert: edit.insert,
    })),
    ...(action.select ? { selection: { anchor: action.select.to, head: action.select.from } } : {}),
    effects: setActionMenu.of(null),
    userEvent: "refactor",
    scrollIntoView: true,
  });
  // Rename's panel takes focus for the new name; without one the editor keeps it.
  if (!action.select || !startRename(view, action.select.from)) view.focus();
}

function renderMenu(view: EditorView, menu: NonNullable<ActionMenu>) {
  const dom = document.createElement("div");
  dom.className = "cm-tariff-actions";
  const list = document.createElement("ul");
  list.setAttribute("role", "menu");
  list.setAttribute("aria-label", translate("Refactorings"));
  menu.actions.forEach((action, index) => {
    const item = document.createElement("li");
    item.setAttribute("role", "none");
    const button = document.createElement("button");
    button.type = "button";
    button.setAttribute("role", "menuitem");
    button.className = "cm-tariff-action";
    if (index === menu.active) button.setAttribute("aria-current", "true");
    // A title names the literal in backticks; it is set as code, as a doc's spans are.
    appendTariffDoc(button, action.title);
    button.addEventListener("mousedown", (event) => event.preventDefault());
    button.addEventListener("click", () => applyTariffAction(view, action));
    item.appendChild(button);
    list.appendChild(item);
  });
  dom.appendChild(list);
  return { dom };
}

/**
 * Promote to input: shows the refactorings for the selection in a menu at the cursor, or says
 * there is nothing to promote. One offer is still shown, so the person sees what will change
 * before it does.
 */
export function promoteToInput(view: EditorView): boolean {
  if (view.state.readOnly) return false;
  const actions = actionsAtSelection(view);
  if (actions.length === 0) {
    showEditorNotice(view, translate("Nothing here can be promoted to an input"));
    return true;
  }
  view.dispatch({
    effects: setActionMenu.of({ pos: view.state.selection.main.from, actions, active: 0 }),
  });
  return true;
}

function moveActive(view: EditorView, by: number): boolean {
  const menu = view.state.field(actionMenuField);
  if (!menu) return false;
  const count = menu.actions.length;
  view.dispatch({
    effects: setActionMenu.of({ ...menu, active: (menu.active + by + count) % count }),
  });
  return true;
}

/** The menu's state and its keys: up and down move, Enter applies, Escape closes. */
export function tariffActionMenu(): Extension {
  return [
    actionMenuField,
    keymap.of([{ key: PROMOTE_KEY, run: promoteToInput }]),
    Prec.highest(
      keymap.of([
        { key: "ArrowDown", run: (view) => moveActive(view, 1) },
        { key: "ArrowUp", run: (view) => moveActive(view, -1) },
        {
          key: "Enter",
          run: (view) => {
            const menu = view.state.field(actionMenuField);
            const action = menu?.actions[menu.active];
            if (!action) return false;
            applyTariffAction(view, action);
            return true;
          },
        },
        {
          key: "Escape",
          run: (view) => {
            if (!view.state.field(actionMenuField)) return false;
            view.dispatch({ effects: setActionMenu.of(null) });
            return true;
          },
        },
      ]),
    ),
  ];
}
