import type { EditorView } from "@codemirror/view";
import { formulaFormat, type FormulaFormatResult } from "@trenova/shared/rules/formula";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";

/** The key that formats the document, as VS Code binds it. */
export const FORMAT_KEY = "Shift-Alt-f";

/**
 * Formats the whole document in place with the rules module's formatter,
 * keeping the cursor on the same line where it can. A formula the formatter
 * refuses (empty, not parsing, holding a comment) is left exactly as written.
 */
export function formatEditorDocument(view: EditorView, rules: RulesRuntime): FormulaFormatResult {
  const source = view.state.doc.toString();
  const result = formulaFormat(rules, { source });
  if (!result.ok || !result.changed) return result;
  const line = view.state.doc.lineAt(view.state.selection.main.head).number;
  view.dispatch({
    changes: { from: 0, to: source.length, insert: result.text },
    userEvent: "format",
  });
  const lines = view.state.doc.lines;
  const target = view.state.doc.line(Math.min(line, lines));
  view.dispatch({ selection: { anchor: target.to } });
  return result;
}
