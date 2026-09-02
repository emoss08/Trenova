import type { EditorView } from "@codemirror/view";

/**
 * Inserts a snippet at the current cursor, replacing any selection. When
 * cursorOffset is given the caret lands that many characters into the snippet
 * (e.g. inside the parentheses of a function call); otherwise it lands at the
 * end of the inserted text.
 */
export function insertSnippet(view: EditorView, text: string, cursorOffset?: number): void {
  const { from, to } = view.state.selection.main;
  const anchor = from + (cursorOffset ?? text.length);

  view.dispatch({
    changes: { from, to, insert: text },
    selection: { anchor },
    scrollIntoView: true,
  });
  view.focus();
}

export function functionSnippet(name: string): { text: string; cursorOffset: number } {
  const text = `${name}()`;
  return { text, cursorOffset: name.length + 1 };
}
