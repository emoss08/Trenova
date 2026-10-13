import {
  checkExpression,
  engineErrorRange,
  engineErrorText,
  type ExprDiagnostic,
  type ExprType,
} from "@/components/formula-editor/expr-check";
import type { Program } from "@/components/formula-editor/expr-parser";
import type { KnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import type { FormulaStatement } from "@trenova/shared/types/formula-template";

export { inferInputType } from "@/components/formula-editor/expr-check";

export type ExpressionAnalysis = {
  empty: boolean;
  /** Everything the language service found, errors first by position. */
  problems: ExprDiagnostic[];
  /** Problems that stop the expression from running. */
  errorCount: number;
  warningCount: number;
  /** Shipment schema variables the expression reads, by root path. */
  schemaReads: ReadonlySet<string>;
  /** The template's own inputs the expression reads. */
  inputReads: ReadonlySet<string>;
  functionCalls: ReadonlySet<string>;
  /** Every name the expression reads, inputs first, in the order they first appear. */
  reads: readonly string[];
  /** Names bound with `let`; they are the expression's own and never inputs. */
  bindings: ReadonlySet<string>;
  program: Program;
  resultType: ExprType;
};

const EMPTY_SET: ReadonlySet<string> = new Set();

/**
 * What the expression reads and what is wrong with it, from the same
 * language service the editor underlines with, so the problems list, the
 * status bar and the rate trace never disagree with the squiggles.
 */
export function analyzeExpression(expression: string, known: KnownIdentifiers): ExpressionAnalysis {
  const analysis = checkExpression(expression, known);
  const empty = analysis.program.empty;
  const problems = empty ? [] : analysis.diagnostics;
  return {
    empty,
    problems,
    errorCount: problems.filter((problem) => problem.severity === "error").length,
    warningCount: problems.filter((problem) => problem.severity === "warning").length,
    schemaReads: empty ? EMPTY_SET : analysis.schemaReads,
    inputReads: empty ? EMPTY_SET : analysis.inputReads,
    functionCalls: empty ? EMPTY_SET : analysis.functionCalls,
    reads: empty ? [] : analysis.reads,
    bindings: new Set(analysis.bindings.keys()),
    program: analysis.program,
    resultType: analysis.resultType,
  };
}

/**
 * What the engine said when it ran the expression on the preview load, as a
 * problem anchored where it happened: on the statement the trace blames, or
 * at the position expr writes into its message, or on the whole expression.
 */
export function engineProblem(
  expression: string,
  error: string,
  statements: readonly FormulaStatement[] | null | undefined,
): ExprDiagnostic {
  const failed = statements?.find((statement) => statement.error);
  const text = engineErrorText(failed?.error ?? error);
  const range = failed
    ? { from: failed.start, to: failed.end }
    : (engineErrorRange(error, expression) ?? { from: 0, to: expression.length });
  return {
    ...range,
    severity: "error",
    code: "engine",
    message: text,
    fixes: [],
  };
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
