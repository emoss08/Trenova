// lading-lint.ts draws `lading.check`'s findings for the file being edited: the underline, the
// translated message, and each fix as an action that applies the module's own edits.

import { linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import type { EditorView } from "@codemirror/view";
import type { LadingDiagnostic, LadingFix } from "@trenova/shared/edi/lading";
import { translateMessage } from "@trenova/shared/i18n/message";
import { cursorTooltip } from "../formula-editor/cursor-tooltip";
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

/** A finding as CodeMirror draws it, clamped to the text it is about. */
export function toLadingLint(diagnostic: LadingDiagnostic, length: number): Diagnostic {
  const actions: Action[] = diagnostic.fixes.map((fix) => ({
    name: translateMessage(fix.label),
    apply: (view) => applyLadingFix(view, fix),
  }));
  const from = Math.min(diagnostic.range.from, length);
  return {
    from,
    to: Math.min(Math.max(diagnostic.range.to, from), length),
    severity: diagnostic.severity,
    source: diagnostic.code,
    message: translateMessage(diagnostic.message),
    ...(actions.length > 0 ? { actions } : {}),
  };
}

/** The linter: the module checks the project 250 ms after the last edit. */
export function ladingLinter(service: LadingLanguageService): Extension {
  return [
    cursorTooltip,
    linter(
      (view): Diagnostic[] => {
        const length = view.state.doc.length;
        return (checkedFile(service)?.diagnostics ?? []).map((diagnostic) =>
          toLadingLint(diagnostic, length),
        );
      },
      { delay: 250 },
    ),
  ];
}
