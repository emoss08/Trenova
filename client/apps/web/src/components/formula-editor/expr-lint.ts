import { linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import { translateMessage } from "@trenova/shared/i18n/message";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type {
  FormulaDiagnostic,
  FormulaFix,
  FormulaInputType,
} from "@trenova/shared/types/formula-template";
import { checkExpression } from "./formula-check";
import type { KnownIdentifiers } from "./known-identifiers";

export type DeclareInput = (name: string, type: FormulaInputType) => void;

/**
 * What the editor underlines: every finding of `formula.check` but the one that
 * says the formula is empty, which an empty editor shows by being empty.
 */
export function lintFindings(diagnostics: readonly FormulaDiagnostic[]): FormulaDiagnostic[] {
  return diagnostics.filter((diagnostic) => diagnostic.code !== "empty");
}

function toAction(fix: FormulaFix, declare: DeclareInput | undefined): Action | null {
  if (fix.kind === "replace") {
    return {
      name: translateMessage(fix.label),
      apply: (view) => {
        view.dispatch({ changes: fix.changes, userEvent: "fix" });
        view.focus();
      },
    };
  }
  if (!declare) return null;
  return {
    name: translateMessage(fix.label),
    apply: () => declare(fix.name, fix.type),
  };
}

/** Turns a finding into what CodeMirror draws: the underline, its message and its fixes. */
export function toCodeMirrorDiagnostic(
  diagnostic: FormulaDiagnostic,
  declare?: DeclareInput,
): Diagnostic {
  const actions = diagnostic.fixes
    .map((fix) => toAction(fix, declare))
    .filter((action): action is Action => action !== null);
  return {
    from: diagnostic.from,
    to: Math.max(diagnostic.to, diagnostic.from),
    severity: diagnostic.severity,
    message: translateMessage(diagnostic),
    source: diagnostic.code,
    ...(actions.length > 0 ? { actions } : {}),
  };
}

/**
 * The linter: the rules module checks the document 300 ms after the last edit,
 * on the main thread, against what the editor knows at that moment.
 */
export function createExprLinter(
  getRules: () => RulesRuntime,
  getKnown: () => KnownIdentifiers,
  getDeclare?: () => DeclareInput | undefined,
): Extension {
  return linter(
    (view): Diagnostic[] => {
      const check = checkExpression(getRules(), view.state.doc.toString(), getKnown());
      return lintFindings(check.diagnostics).map((diagnostic) =>
        toCodeMirrorDiagnostic(diagnostic, getDeclare?.()),
      );
    },
    { delay: 300 },
  );
}
