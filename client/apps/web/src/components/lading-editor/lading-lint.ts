// lading-lint.ts draws `lading.check`'s findings for the file being edited, checked with the text
// as the editor holds it: the underline, the translated message with its code, the places it
// relates to (each one a link that moves there), and each fix as an action that applies the
// module's own edits. F8 and Shift-F8 move between findings, as in VS Code, and Mod-Shift-M
// opens them all in a panel.

import { linter, openLintPanel, type Action, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import type { LadingDiagnostic, LadingFix } from "@trenova/shared/edi/lading";
import { translateMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import { cursorTooltip } from "../formula-editor/cursor-tooltip";
import { jumpToDiagnostic } from "../formula-editor/tariff-lint";
import { checkedFile, type LadingLanguageService } from "./lading-service";

/** Applies a fix's edits, which the module guarantees do not overlap. */
export function applyLadingFix(view: EditorView, fix: Pick<LadingFix, "changes">) {
  view.dispatch({
    changes: fix.changes.map((change) => ({
      from: change.range.from,
      to: change.range.to,
      insert: change.insert,
    })),
    userEvent: "fix",
  });
  view.focus();
}

function clamp(range: { from: number; to: number }, length: number) {
  const from = Math.min(range.from, length);
  return { from, to: Math.min(Math.max(range.to, from), length) };
}

/** The message, its code, and the places it relates to as links. */
function renderLadingMessage(diagnostic: LadingDiagnostic, message: string) {
  return (view: EditorView): Node => {
    const dom = document.createElement("div");
    dom.className = "cm-lading-message";
    const text = document.createElement("span");
    text.textContent = message;
    dom.appendChild(text);
    const code = document.createElement("span");
    code.className = "cm-lading-message-code";
    code.textContent = diagnostic.code;
    dom.appendChild(code);
    for (const related of diagnostic.related) {
      const link = document.createElement("button");
      link.type = "button";
      link.className = "cm-lading-message-related";
      const line = view.state.doc.lineAt(Math.min(related.range.from, view.state.doc.length));
      link.textContent = translate("Line {0}: {1}", line.number, translateMessage(related.message));
      link.addEventListener("mousedown", (event) => {
        event.preventDefault();
        const range = clamp(related.range, view.state.doc.length);
        view.dispatch({
          selection: { anchor: range.from, head: range.to },
          effects: EditorView.scrollIntoView(range.from, { y: "center" }),
        });
        view.focus();
      });
      dom.appendChild(link);
    }
    return dom;
  };
}

/** A finding as CodeMirror draws it, clamped to the text it is about. */
export function toLadingLint(diagnostic: LadingDiagnostic, length: number): Diagnostic {
  const actions: Action[] = diagnostic.fixes.map((fix) => ({
    name: translateMessage(fix.label),
    apply: (view) => applyLadingFix(view, fix),
  }));
  const message = translateMessage(diagnostic.message);
  return {
    ...clamp(diagnostic.range, length),
    severity: diagnostic.severity,
    source: diagnostic.code,
    message,
    renderMessage: renderLadingMessage(diagnostic, message),
    ...(actions.length > 0 ? { actions } : {}),
  };
}

/**
 * The linter: the module checks the project, with this file as the editor holds it, 250 ms after
 * the last edit. An empty file is not underlined: there is nothing wrong with it yet.
 */
export function ladingLinter(service: LadingLanguageService): Extension {
  return [
    cursorTooltip,
    linter(
      (view): Diagnostic[] => {
        const source = view.state.doc.toString();
        if (source.trim() === "") return [];
        return (checkedFile(service, source)?.diagnostics ?? []).map((diagnostic) =>
          toLadingLint(diagnostic, source.length),
        );
      },
      { delay: 250 },
    ),
    keymap.of([
      { key: "F8", run: (view) => jumpToDiagnostic(view, 1) },
      { key: "Shift-F8", run: (view) => jumpToDiagnostic(view, -1) },
      { key: "Mod-Shift-m", run: openLintPanel },
    ]),
    EditorView.baseTheme({
      ".cm-lading-message": { display: "flex", flexDirection: "column", gap: "2px" },
      ".cm-lading-message-code": {
        fontFamily: "var(--font-code)",
        fontSize: "11px",
        color: "var(--foreground-subtle)",
      },
      ".cm-lading-message-related": {
        all: "unset",
        cursor: "pointer",
        fontSize: "11px",
        color: "var(--foreground-muted)",
        textDecoration: "underline dotted",
        textUnderlineOffset: "3px",
      },
    }),
  ];
}
