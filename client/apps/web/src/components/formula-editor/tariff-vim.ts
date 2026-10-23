// tariff-vim.ts is Vim mode for the Tariff editor, with AstroNvim's defaults on top of Vim's own
// keys: `jj` or `jk` leaves insert mode, Space is the leader, and the language features answer
// to the keys Neovim's LSP binds them to. It is loaded only when the person turns Vim keys on
// (`useTariffVim`), so nobody else downloads it. Vim's key mappings are global to the page, so
// they are defined once, the first time the module is asked for the extension.

import { toggleLineComment } from "@codemirror/commands";
import { openLintPanel } from "@codemirror/lint";
import { Prec, type Extension } from "@codemirror/state";
import type { Command, EditorView } from "@codemirror/view";
import { CodeMirror, Vim, vim } from "@replit/codemirror-vim";
import { requestEditorSave, tariffEditorCommands } from "./tariff-commands";
import { showHoverAtCursor } from "./tariff-hover";
import { jumpToDiagnostic, showLineDiagnostics } from "./tariff-lint";
import { goToDefinitionAtCursor, startRename } from "./tariff-navigate";
import { findReferences } from "./tariff-references";

/** How long, in ms, `jj` and `jk` wait for their second key: AstroNvim's `timeoutlen`. */
const ESCAPE_KEYS_TIMEOUT = 500;

const format: Command = (view) => view.state.facet(tariffEditorCommands)?.format(view) ?? false;

/**
 * Runs a command that selects what it lands on and leaves the cursor at its start instead, as
 * Neovim does: a selection would put Vim in visual mode.
 */
const landing =
  (run: Command): Command =>
  (view) => {
    if (!run(view)) return false;
    const { from, empty } = view.state.selection.main;
    if (!empty) view.dispatch({ selection: { anchor: from } });
    return true;
  };

/** Normal-mode keys and the command each runs, as AstroNvim binds them. */
const TARIFF_VIM_NORMAL_KEYS: ReadonlyArray<readonly [keys: string, run: Command]> = [
  ["gd", landing(goToDefinitionAtCursor)],
  ["K", showHoverAtCursor],
  ["gl", showLineDiagnostics],
  ["<Space>ld", showLineDiagnostics],
  ["]d", (view) => jumpToDiagnostic(view, 1)],
  ["[d", (view) => jumpToDiagnostic(view, -1)],
  ["]e", (view) => jumpToDiagnostic(view, 1, "error")],
  ["[e", (view) => jumpToDiagnostic(view, -1, "error")],
  ["]w", (view) => jumpToDiagnostic(view, 1, "warning")],
  ["[w", (view) => jumpToDiagnostic(view, -1, "warning")],
  ["<Space>la", openLintPanel],
  ["gra", openLintPanel],
  ["<Space>lD", openLintPanel],
  ["<Space>lf", format],
  ["<Space>lr", startRename],
  ["grn", startRename],
  ["<Space>lR", findReferences],
  ["grr", findReferences],
  ["<Space>/", toggleLineComment],
  ["<Space>w", requestEditorSave],
];

let configured = false;

function configure() {
  if (configured) return;
  configured = true;

  Vim.setOption("insertModeEscKeysTimeout", ESCAPE_KEYS_TIMEOUT);
  Vim.map("jj", "<Esc>", "insert");
  Vim.map("jk", "<Esc>", "insert");

  // Space moves right in Vim; AstroNvim makes it the leader. The built-in mapping has no mode,
  // and `unmap` removes it only when asked for none.
  (Vim.unmap as (lhs: string, ctx?: string) => boolean)("<Space>");

  // `:w` runs the adapter's save command.
  CodeMirror.commands.save = (cm: CodeMirror) => requestEditorSave(cm.cm6);

  TARIFF_VIM_NORMAL_KEYS.forEach(([keys, run], index) => {
    const name = `tariff${index}`;
    Vim.defineAction(name, (cm) => {
      const view: EditorView = cm.cm6;
      run(view);
    });
    Vim.mapCommand(keys, "action", name, {}, { context: "normal" });
  });

  Vim.defineAction("tariffToggleComment", (cm) => {
    const view: EditorView = cm.cm6;
    toggleLineComment(view);
    Vim.exitVisualMode(cm);
  });
  Vim.mapCommand("<Space>/", "action", "tariffToggleComment", {}, { context: "visual" });
}

let extension: Extension | null = null;

/** Vim mode with its status line (the mode, and the `:` and `/` prompts) under the text. */
export function tariffVim(): Extension {
  configure();
  extension ??= Prec.highest(vim({ status: true }));
  return extension;
}
