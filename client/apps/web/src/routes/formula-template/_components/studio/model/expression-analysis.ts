import { checkExpression } from "@/components/formula-editor/formula-check";
import { lintFindings } from "@/components/formula-editor/expr-lint";
import type { KnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import type {
  FormulaBinding,
  FormulaCheckStatement,
  FormulaLookupTable,
} from "@trenova/shared/rules/formula";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type { FormulaDiagnostic, FormulaValueType } from "@trenova/shared/types/formula-template";

export type ExpressionAnalysis = {
  /** Nothing but spaces and comments: the studio shows where to start, not a problem. */
  empty: boolean;
  /** Everything `formula.check` found, sorted by position, except that the formula is empty. */
  problems: FormulaDiagnostic[];
  /** Problems that stop the expression from saving. */
  errorCount: number;
  warningCount: number;
  /** Schema fields the expression reads, by the path it reads them as. */
  schemaReads: ReadonlySet<string>;
  /** The template's own inputs the expression reads. */
  inputReads: ReadonlySet<string>;
  functionCalls: ReadonlySet<string>;
  /** Every name the expression reads, inputs first, in the order they first appear. */
  reads: readonly string[];
  /** The top-level `let` lines; they are the expression's own and never inputs. */
  bindings: readonly FormulaBinding[];
  /** Each `let` line and value, with its range: what the trace and the inline values follow. */
  statements: readonly FormulaCheckStatement[];
  /** Every rate table the expression reads, in source order. */
  lookupTables: readonly FormulaLookupTable[];
  resultType: FormulaValueType;
  /** The schema is still loading, so names have not been checked yet. */
  pending: boolean;
};

/**
 * What the expression reads and what is wrong with it, from the same
 * `formula.check` the editor underlines with and the server saves by, so the
 * problems list, the status bar and the rate trace never disagree with the
 * squiggles.
 */
export function analyzeExpression(
  rules: RulesRuntime,
  expression: string,
  known: KnownIdentifiers,
): ExpressionAnalysis {
  const check = checkExpression(rules, expression, known);
  const problems = lintFindings(check.diagnostics);
  return {
    empty: check.empty,
    problems,
    errorCount: problems.filter((problem) => problem.severity === "error").length,
    warningCount: problems.filter((problem) => problem.severity === "warning").length,
    schemaReads: new Set(check.schemaReads),
    inputReads: new Set(check.inputReads),
    functionCalls: new Set(check.functionCalls),
    reads: check.reads,
    bindings: check.bindings,
    statements: check.statements,
    lookupTables: check.lookupTables,
    resultType: check.resultType,
    pending: check.pending,
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
