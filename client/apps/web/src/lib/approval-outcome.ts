import { toast } from "@trenova/shared/components/ui/toast";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * What deciding one proposal came to, as `AgentProposalDecisionResult`
 * carries it. `error` is set when the decision was recorded but its write did
 * not go through: `refused` when the record it changes moved on since it was
 * proposed, so nothing was changed, else the write failed when it ran.
 */
export type ProposalDecisionOutcome = {
  refused: boolean;
  executed: boolean;
  error?: string | null;
};

/** Why a recorded approval's write did not go through; null when it ran or waits to. */
export function approvalFailure(outcome: ProposalDecisionOutcome): string | null {
  const error = outcome.error?.trim() ?? "";
  return outcome.refused || error !== "" ? error : null;
}

/**
 * Says that an approval was refused because what it changes moved on since
 * the agent proposed it. The server recorded the approval, ran nothing and
 * started a turn in the conversation that raised it; `why` is the server's
 * explanation of what moved, which says that nothing was changed.
 *
 * The card the person approved has gone by the time the answer comes back,
 * so this is news the screen cannot show on its own.
 */
export function announceRefusal(t: TranslateFn, why: string | null | undefined) {
  toast.info(t("This changed since you opened it"), {
    description: why?.trim() || t("Nothing was changed."),
  });
}
