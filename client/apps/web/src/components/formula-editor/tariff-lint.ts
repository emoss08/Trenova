// tariff-lint.ts draws `tariff.check`'s findings: the underline, the translated message, and
// each fix as an action that applies the module's own source edits. A finding inside an
// imported library is shown on the `use` line that brings that library in, since its range is
// in the library's source; a finding about a contract input value has no place in the source
// and is left to the form that holds the value.

import { linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import type { EditorView } from "@codemirror/view";
import { translateMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TariffDiagnostic, TariffFix } from "@trenova/shared/types/tariff";
import { checkService, type TariffLanguageService } from "./tariff-service";

/** Applies a fix's edits, which the module guarantees do not overlap. */
export function applyTariffFix(view: EditorView, fix: TariffFix) {
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

/** Where a library's finding is shown: the `use` of that module, or the first line. */
function importRange(source: string, module: string): { from: number; to: number } {
  const at = source.indexOf(module);
  if (at >= 0) return { from: at, to: at + module.length };
  const end = source.indexOf("\n");
  return { from: 0, to: end === -1 ? source.length : end };
}

/** A finding as CodeMirror draws it, or null for one with no place in this source. */
export function toTariffLint(diagnostic: TariffDiagnostic, source: string): Diagnostic | null {
  const message = translateMessage(diagnostic.message);
  if (diagnostic.module) {
    const range = importRange(source, diagnostic.module);
    return {
      ...range,
      severity: diagnostic.severity,
      source: diagnostic.code,
      message: translate("In {0}: {1}", diagnostic.module, message),
    };
  }
  if (!diagnostic.range) return null;
  const actions: Action[] = diagnostic.fixes.map((fix) => ({
    name: translateMessage(fix.label),
    apply: (view) => applyTariffFix(view, fix),
  }));
  const length = source.length;
  const from = Math.min(diagnostic.range.from, length);
  return {
    from,
    to: Math.min(Math.max(diagnostic.range.to, from), length),
    severity: diagnostic.severity,
    source: diagnostic.code,
    message,
    ...(actions.length > 0 ? { actions } : {}),
  };
}

/** The linter: the module checks the document 250 ms after the last edit. */
export function tariffLinter(service: TariffLanguageService): Extension {
  return linter(
    (view): Diagnostic[] => {
      const source = view.state.doc.toString();
      return checkService(service, source)
        .diagnostics.map((diagnostic) => toTariffLint(diagnostic, source))
        .filter((diagnostic): diagnostic is Diagnostic => diagnostic !== null);
    },
    { delay: 250 },
  );
}
