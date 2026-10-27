// lading-assist.ts is the Lading editor's navigation and layout, answered by the EDI module: go to
// definition (`lading.definition`, F12 or Mod-click, into another file of the project), folding
// from the check's outline, and formatting (`lading.format`, Shift-Alt-F as VS Code binds it).

import { foldService } from "@codemirror/language";
import { Prec, type Extension } from "@codemirror/state";
import { EditorView, keymap, type Command } from "@codemirror/view";
import { ladingDefinition, ladingFormat } from "@trenova/shared/edi/lading";
import { translateMessage } from "@trenova/shared/i18n/message";
import { tariffEditorCommands } from "../formula-editor/tariff-commands";
import { foldAt } from "../formula-editor/tariff-fold";
import { FORMAT_KEY } from "../formula-editor/tariff-format";
import {
  checkedFile,
  ladingRequest,
  ladingServiceFacet,
  unlessRefused,
  type LadingLanguageService,
} from "./lading-service";

function goToDefinitionAt(view: EditorView, position: number): boolean {
  const service = view.state.facet(ladingServiceFacet);
  if (!service) return false;
  const targets = unlessRefused(
    () =>
      ladingDefinition(service.edi(), ladingRequest(service, view.state.doc.toString(), position)),
    [],
  );
  const target = targets[0];
  if (!target) return false;
  service.openLocation(target);
  return true;
}

/** Goes to the declaration of the name at the cursor; false when there is none. */
export const goToLadingDefinition: Command = (view) =>
  goToDefinitionAt(view, view.state.selection.main.head);

/**
 * Formats the document in place, keeping the cursor on its line; false (and the reason) when the
 * file has syntax errors.
 */
export function formatLadingDocument(
  view: EditorView,
  service: LadingLanguageService,
): { formatted: boolean; reason: string | null } {
  const source = view.state.doc.toString();
  const result = unlessRefused(() => ladingFormat(service.edi(), source), null);
  if (!result) return { formatted: false, reason: null };
  if ("error" in result) return { formatted: false, reason: translateMessage(result.error) };
  if (result.text !== source) {
    const line = view.state.doc.lineAt(view.state.selection.main.head);
    const column = view.state.selection.main.head - line.from;
    view.dispatch({
      changes: { from: 0, to: source.length, insert: result.text },
      userEvent: "format",
    });
    const target = view.state.doc.line(Math.min(line.number, view.state.doc.lines));
    view.dispatch({ selection: { anchor: Math.min(target.from + column, target.to) } });
  }
  return { formatted: true, reason: null };
}

/** Moves the cursor to a range of this file and shows it. */
export function revealLadingRange(view: EditorView, range: { from: number; to: number }) {
  const to = Math.min(range.to, view.state.doc.length);
  const from = Math.min(range.from, to);
  view.dispatch({
    selection: { anchor: from, head: to },
    effects: EditorView.scrollIntoView(from, { y: "center" }),
  });
  view.focus();
}

export function ladingAssist(service: LadingLanguageService): Extension {
  const format: Command = (view) => view.state.facet(tariffEditorCommands)?.format(view) ?? false;
  return [
    ladingServiceFacet.of(service),
    foldService.of((state, lineStart, lineEnd) => {
      const file = checkedFile(service, state.doc.toString());
      return file ? foldAt(file.folds, lineStart, lineEnd) : null;
    }),
    EditorView.domEventHandlers({
      mousedown(event, view) {
        if (!(event.metaKey || event.ctrlKey) || event.button !== 0) return false;
        const position = view.posAtCoords({ x: event.clientX, y: event.clientY });
        if (position === null || !goToDefinitionAt(view, position)) return false;
        event.preventDefault();
        return true;
      },
    }),
    Prec.high(
      keymap.of([
        { key: "F12", run: goToLadingDefinition },
        { key: FORMAT_KEY, run: format },
      ]),
    ),
  ];
}
