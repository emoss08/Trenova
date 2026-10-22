import type { TariffTestsState } from "@/components/tariff-studio/use-tariff-tests";
import type { TariffCheckResult, TariffDiagnostic } from "@trenova/shared/types/tariff";

export type ProposalCheck =
  /** The browser's check, against this Studio's context, finds an error. */
  | { kind: "problem"; problem: TariffDiagnostic }
  /** It checks, but some of its tests fail. */
  | { kind: "failing"; failed: number; total: number }
  | { kind: "ok" };

/**
 * Where a proposal stands in this Studio. The server offers one only when it checks clean and
 * its tests pass, but the browser checks it again against the context this Studio has (which
 * can differ: a code archived since, a library changed), then runs its tests.
 */
export function proposalCheck(check: TariffCheckResult, tests: TariffTestsState): ProposalCheck {
  const problem = check.diagnostics.find((finding) => finding.severity === "error");
  if (problem) return { kind: "problem", problem };
  if (tests.status === "ran" && !tests.stale && tests.failed > 0) {
    return { kind: "failing", failed: tests.failed, total: tests.passed + tests.failed };
  }
  return { kind: "ok" };
}
