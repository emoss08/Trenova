import { linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import { checkExpression, type ExprDiagnostic, type ExprFix } from "./expr-check";
import { tokenize } from "./expr-parser";
import type { KnownIdentifiers } from "./known-identifiers";

export type { ExprDiagnostic, ExprFix } from "./expr-check";

export type IdentifierToken = {
  name: string;
  from: number;
  to: number;
  isCall: boolean;
};

/**
 * The identifiers an expression reads, dotted paths joined, with whether each
 * is called. Built on the language's own tokenizer, so strings and comments
 * never count.
 */
export function collectIdentifierTokens(expression: string): IdentifierToken[] {
  const { tokens } = tokenize(expression);
  const out: IdentifierToken[] = [];
  let index = 0;
  while (index < tokens.length) {
    const token = tokens[index];
    if (token.kind !== "ident") {
      index += 1;
      continue;
    }
    let name = token.text;
    let to = token.to;
    let cursor = index + 1;
    while (
      tokens[cursor]?.kind === "punct" &&
      tokens[cursor].text === "." &&
      tokens[cursor].to === tokens[cursor + 1]?.from &&
      (tokens[cursor + 1]?.kind === "ident" || tokens[cursor + 1]?.kind === "keyword")
    ) {
      name += `.${tokens[cursor + 1].text}`;
      to = tokens[cursor + 1].to;
      cursor += 2;
    }
    const next = tokens[cursor];
    out.push({
      name,
      from: token.from,
      to,
      isCall: next?.kind === "punct" && next.text === "(",
    });
    index = cursor;
  }
  return out;
}

const LET_BINDING = /\blet\s+([A-Za-z_][A-Za-z0-9_]*)\s*=/g;

/** Names the expression binds itself with `let`; they are known wherever they are read. */
export function letBindings(expression: string): Set<string> {
  const names = new Set<string>();
  for (const match of expression.matchAll(LET_BINDING)) {
    names.add(match[1]);
  }
  return names;
}

/** Everything the language service finds wrong with the expression. */
export function lintExpression(expression: string, known: KnownIdentifiers): ExprDiagnostic[] {
  if (expression.trim() === "") return [];
  return checkExpression(expression, known).diagnostics;
}

export type DeclareInput = (name: string, type: "Number" | "String" | "Boolean") => void;

function toAction(fix: ExprFix, declare: DeclareInput | undefined): Action | null {
  if (fix.kind === "replace") {
    return {
      name: fix.label,
      apply: (view) => {
        view.dispatch({ changes: fix.changes, userEvent: "fix" });
        view.focus();
      },
    };
  }
  if (!declare) return null;
  return {
    name: fix.label,
    apply: () => declare(fix.name, fix.type),
  };
}

/** Turns a finding into what CodeMirror draws: the underline, its message and its fixes. */
export function toCodeMirrorDiagnostic(
  diagnostic: ExprDiagnostic,
  declare?: DeclareInput,
): Diagnostic {
  const actions = diagnostic.fixes
    .map((fix) => toAction(fix, declare))
    .filter((action): action is Action => action !== null);
  return {
    from: diagnostic.from,
    to: Math.max(diagnostic.to, diagnostic.from),
    severity: diagnostic.severity === "info" ? "info" : diagnostic.severity,
    message: diagnostic.message,
    source: diagnostic.code,
    ...(actions.length > 0 ? { actions } : {}),
  };
}

export function createExprLinter(
  getKnown: () => KnownIdentifiers,
  getDeclare?: () => DeclareInput | undefined,
): Extension {
  return linter(
    (view): Diagnostic[] =>
      lintExpression(view.state.doc.toString(), getKnown()).map((diagnostic) =>
        toCodeMirrorDiagnostic(diagnostic, getDeclare?.()),
      ),
    { delay: 300 },
  );
}
