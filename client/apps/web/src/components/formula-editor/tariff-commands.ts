// tariff-commands.ts holds what the editor's key bindings share without importing each other:
// the commands the editor owns (format, so the keyboard and Vim report a refusal the same way, and
// the language commands of an editor whose language is not Tariff), the request to save the page
// the editor is on, and the comment syntax line commenting reads.

import { EditorState, Facet } from "@codemirror/state";
import type { Command, EditorView } from "@codemirror/view";

export type TariffEditorCommands = {
  /** Formats the document and tells the page what happened. */
  format: Command;
  /**
   * The language commands Vim's keys run. An editor of another language (Lading) gives its own;
   * left out, they are Tariff's.
   */
  definition?: Command;
  hover?: Command;
  lineProblems?: Command;
  rename?: Command;
  references?: Command;
  actions?: Command;
  /** Shows every finding in the project; the host's own problems list, as the Lading studio's. */
  problems?: Command;
};

export const tariffEditorCommands = Facet.define<TariffEditorCommands, TariffEditorCommands | null>(
  {
    combine: (values) => values[0] ?? null,
  },
);

/** The event an editor raises to save the page it is on; the Studios answer it as Ctrl+S. */
export const EDITOR_SAVE_EVENT = "trenova:editor-save";

/** Asks the page holding the editor to save, as if the person had pressed Ctrl+S. */
export function requestEditorSave(view: EditorView): boolean {
  view.dom.dispatchEvent(new CustomEvent(EDITOR_SAVE_EVENT, { bubbles: true }));
  return true;
}

/** Tariff's `//` line comments, for toggle comment (Ctrl+/, and `gc` in Vim). */
export const tariffLanguageData = EditorState.languageData.of(() => [
  { commentTokens: { line: "//" } },
]);
