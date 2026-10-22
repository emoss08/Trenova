import type { FormulaDiagnostic } from "@trenova/shared/types/formula-template";
import type { ExpressionAnalysis } from "../model/expression-analysis";

export type ProposalCheck =
  | { kind: "engine"; message: string }
  | { kind: "problem"; problem: FormulaDiagnostic }
  | { kind: "pending" }
  | { kind: "ok" };

type EngineCheck = { valid: boolean; error: string };
type NameAnalysis = Pick<ExpressionAnalysis, "pending" | "problems">;

/**
 * Where a proposal stands with the checks that ran on it: the engine's own
 * run first, then the rules module's check in the browser, which can only
 * vouch for names once the schema has arrived. A syntax error needs no
 * schema, so it is reported even while names are still unchecked.
 */
export function proposalCheck(check: EngineCheck, analysis: NameAnalysis): ProposalCheck {
  if (!check.valid) {
    return { kind: "engine", message: check.error };
  }
  const problem = analysis.problems.find((finding) => finding.severity === "error");
  if (problem) {
    return { kind: "problem", problem };
  }
  if (analysis.pending) {
    return { kind: "pending" };
  }
  return { kind: "ok" };
}
