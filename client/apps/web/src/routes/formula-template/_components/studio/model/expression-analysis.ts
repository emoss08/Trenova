import {
  collectIdentifierTokens,
  letBindings,
  lintExpression,
} from "@/components/formula-editor/expr-lint";
import {
  EXPR_KEYWORDS,
  isKnownFunction,
  isKnownVariable,
  isOperatorWord,
  type KnownIdentifiers,
} from "@/components/formula-editor/known-identifiers";
import type { VariableValueType } from "@trenova/shared/types/formula-template";

export type ExpressionProblemKind = "unknown-variable" | "error";

/** One thing wrong with the expression, as the editor's own linter reads it. */
export type ExpressionProblem = {
  kind: ExpressionProblemKind;
  from: number;
  to: number;
  message: string;
  /** The identifier an unknown-variable problem is about. */
  name?: string;
};

export type ExpressionAnalysis = {
  empty: boolean;
  problems: ExpressionProblem[];
  /** Problems that stop the expression from compiling: syntax, unknown names. */
  errorCount: number;
  /** Shipment schema variables the expression reads, by root path. */
  schemaReads: ReadonlySet<string>;
  /** The template's own inputs the expression reads. */
  inputReads: ReadonlySet<string>;
  functionCalls: ReadonlySet<string>;
  /** Every name the expression reads, inputs first, in the order they first appear. */
  reads: readonly string[];
  /** Names bound with `let`; they are the expression's own and never inputs. */
  bindings: ReadonlySet<string>;
};

/**
 * What the expression reads and what is wrong with it, from the same
 * tokenizer and linter the editor draws with, so the problems list, the
 * status bar and the rate trace never disagree with the squiggles.
 */
export function analyzeExpression(
  expression: string,
  known: KnownIdentifiers,
  customNames: ReadonlySet<string>,
): ExpressionAnalysis {
  const empty = expression.trim() === "";
  const bindings = letBindings(expression);
  const problems: ExpressionProblem[] = [];
  const schemaReads = new Set<string>();
  const inputReads = new Set<string>();
  const functionCalls = new Set<string>();
  const order: string[] = [];

  if (!empty) {
    for (const diagnostic of lintExpression(expression, known)) {
      const name = expression.slice(diagnostic.from, diagnostic.to);
      if (diagnostic.severity === "warning") {
        problems.push({
          kind: "unknown-variable",
          from: diagnostic.from,
          to: diagnostic.to,
          message: diagnostic.message,
          name,
        });
      } else {
        problems.push({
          kind: "error",
          from: diagnostic.from,
          to: diagnostic.to,
          message: diagnostic.message,
        });
      }
    }

    for (const token of collectIdentifierTokens(expression)) {
      if (EXPR_KEYWORDS.has(token.name) || isOperatorWord(known, token.name)) continue;
      if (token.isCall) {
        if (isKnownFunction(known, token.name)) functionCalls.add(token.name);
        continue;
      }
      const root = token.name.split(".")[0];
      if (bindings.has(root)) continue;
      if (customNames.has(root)) {
        if (!inputReads.has(root)) order.push(root);
        inputReads.add(root);
      } else if (isKnownVariable(known, token.name)) {
        if (!schemaReads.has(token.name)) order.push(token.name);
        schemaReads.add(token.name);
      }
    }
  }

  const inputsFirst = [
    ...order.filter((name) => inputReads.has(name)),
    ...order.filter((name) => !inputReads.has(name)),
  ];

  return {
    empty,
    problems,
    errorCount: problems.length,
    schemaReads,
    inputReads,
    functionCalls,
    reads: inputsFirst,
    bindings,
  };
}

const BOOLEAN_PREFIX = /^(is|has|requires|uses)[A-Z_]/;

/** The type an input added from an unknown name starts as: a flag for is/has/requires names. */
export function inferInputType(name: string): VariableValueType {
  return BOOLEAN_PREFIX.test(name) ? "Boolean" : "Number";
}

/** A breakdown line's key, followed from its label until the person types their own. */
export function keyFromLabel(label: string): string {
  const words = label
    .replace(/[^A-Za-z0-9 ]+/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  if (words.length === 0) return "";
  const [first, ...rest] = words;
  const key =
    first.charAt(0).toLowerCase() +
    first.slice(1) +
    rest.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join("");
  return /^[0-9]/.test(key) ? `line${key}` : key;
}

/** Strips a typed input name down to what an identifier may hold. */
export function sanitizeIdentifier(raw: string): string {
  return raw.replace(/[^A-Za-z0-9_]/g, "");
}
