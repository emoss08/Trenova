import type { ExpressionAnalysis } from "../model/expression-analysis";

export type ProposalCheck =
  | { kind: "engine"; message: string }
  | { kind: "problem"; message: string }
  | { kind: "pending" }
  | { kind: "ok" };

type EngineCheck = { valid: boolean; error: string };
type NameAnalysis = Pick<ExpressionAnalysis, "pending" | "errorCount" | "problems">;

/**
 * Where a proposal stands with the checks that ran on it: the engine's own
 * run first, then the editor's analysis, which can only vouch for names once
 * the engine's schema has arrived. A syntax error needs no schema, so it is
 * reported even while names are still unchecked.
 */
export function proposalCheck(check: EngineCheck, analysis: NameAnalysis): ProposalCheck {
  if (!check.valid) {
    return { kind: "engine", message: check.error };
  }
  if (analysis.errorCount > 0) {
    return {
      kind: "problem",
      message: analysis.problems.find((problem) => problem.severity === "error")?.message ?? "",
    };
  }
  if (analysis.pending) {
    return { kind: "pending" };
  }
  return { kind: "ok" };
}
