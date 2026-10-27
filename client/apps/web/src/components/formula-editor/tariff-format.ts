// tariff-format.ts formats the document with `tariff.format`, the one canonical style, from the
// keyboard (Shift-Alt-F, as VS Code binds it) or a button. A file with syntax errors is left as
// written, and the caller is told why.

import type { EditorView } from "@codemirror/view";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffFormat, type TariffFormatResult } from "@trenova/shared/rules/tariff";

export const FORMAT_KEY = "Shift-Alt-f";

/** Formats the document in place, keeping the cursor on its line where it can. */
export function formatTariffDocument(view: EditorView, rules: RulesRuntime): TariffFormatResult {
  const source = view.state.doc.toString();
  const result = tariffFormat(rules, source);
  if (result.formatted === null || result.formatted === source) return result;
  const line = view.state.doc.lineAt(view.state.selection.main.head).number;
  view.dispatch({
    changes: { from: 0, to: source.length, insert: result.formatted },
    userEvent: "format",
  });
  const target = view.state.doc.line(Math.min(line, view.state.doc.lines));
  view.dispatch({ selection: { anchor: target.to } });
  return result;
}
