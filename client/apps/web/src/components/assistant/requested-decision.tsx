import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { PlanCard } from "./plan-card";
import { ProposalCard } from "./proposal-card";

/**
 * A waiting proposal the assistant put back in front of the person, because
 * they typed an approval that decides nothing. It is the proposal's own card
 * (or its plan's, for a step of one), read from the conversation's proposals,
 * so approving it here is the same decision as approving it where it was
 * first raised, and it reads as decided in both places once it is.
 */
export function RequestedDecision({
  proposalId,
  threadId,
}: {
  proposalId: string;
  threadId: string;
}) {
  const t = useT();
  const proposalsQuery = useQuery(queries.assistant.proposals(threadId));
  const proposal = useMemo(
    () => proposalsQuery.data?.results.find((candidate) => candidate.id === proposalId) ?? null,
    [proposalsQuery.data, proposalId],
  );
  const planId = proposal?.planId ?? "";
  const plansQuery = useQuery({ ...queries.assistant.plans(threadId), enabled: planId !== "" });
  const plan = useMemo(
    () => plansQuery.data?.results.find((candidate) => candidate.id === planId) ?? null,
    [plansQuery.data, planId],
  );
  const steps = useMemo(
    () =>
      (proposalsQuery.data?.results ?? [])
        .filter((candidate) => planId !== "" && candidate.planId === planId)
        .sort((a, b) => a.planStep - b.planStep),
    [proposalsQuery.data, planId],
  );

  if (proposalsQuery.isPending || (planId !== "" && plansQuery.isPending)) {
    return <Skeleton className="h-24 w-full" aria-label={t("Loading the proposal")} />;
  }

  if (proposal === null) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("This proposal is no longer part of the conversation.")}
      </p>
    );
  }

  if (plan !== null) {
    return <PlanCard plan={plan} steps={steps} threadId={threadId} />;
  }

  return <ProposalCard proposal={proposal} threadId={threadId} />;
}
