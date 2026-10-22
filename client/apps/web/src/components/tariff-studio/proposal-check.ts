import type { TariffTestsState } from "@/components/tariff-studio/use-tariff-tests";
import type { FormulaProposal } from "@/types/page-draft";
import type { TariffCheckResult, TariffDiagnostic } from "@trenova/shared/types/tariff";

export type ProposalCheck =
  /** The server's check refused it before Nova showed it. */
  | { kind: "server"; problem: TariffDiagnostic | null }
  /** The browser's check, against this Studio's context, finds an error. */
  | { kind: "problem"; problem: TariffDiagnostic }
  /** It checks, but some of its tests fail. */
  | { kind: "failing"; failed: number; total: number }
  | { kind: "ok" };

/**
 * Where a proposal stands: the server's check that ran before it was shown first, then the
 * browser's check of the same source against the context this Studio has (which can differ: a
 * code archived since, a library changed), then its tests run in the browser.
 */
export function proposalCheck(
  proposal: Pick<FormulaProposal, "check">,
  check: TariffCheckResult,
  tests: TariffTestsState,
): ProposalCheck {
  if (!proposal.check.ok) {
    return {
      kind: "server",
      problem: proposal.check.diagnostics.find((finding) => finding.severity === "error") ?? null,
    };
  }
  const problem = check.diagnostics.find((finding) => finding.severity === "error");
  if (problem) return { kind: "problem", problem };
  if (tests.status === "ran" && !tests.stale && tests.failed > 0) {
    return { kind: "failing", failed: tests.failed, total: tests.passed + tests.failed };
  }
  return { kind: "ok" };
}
