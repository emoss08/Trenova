import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import type { DecisionRequestRef } from "./decision-requests";
import { PlanCard } from "./plan-card";
import { ProposalBatchBar } from "./proposal-batch";
import { ProposalCard } from "./proposal-card";

/**
 * A waiting decision the assistant put back in front of the person, because
 * they typed an approval that decides nothing. It is the proposal's own card
 * (or its plan's, for a step of one or a plan asked for by id), read from the
 * conversation's proposals, so approving it here is the same decision as
 * approving it where it was first raised, and it reads as decided in both
 * places once it is. Several proposals of one tool asked for together are
 * their cards with one control that approves them all.
 */
export function RequestedDecision({
  request,
  threadId,
}: {
  request: Omit<DecisionRequestRef, "callId">;
  threadId: string;
}) {
  const t = useT();
  const proposalsQuery = useQuery(queries.assistant.proposals(threadId));
  const all = useMemo(() => proposalsQuery.data?.results ?? [], [proposalsQuery.data]);
  const requested = useMemo(
    () =>
      request.proposalIds
        .map((id) => all.find((candidate) => candidate.id === id))
        .filter((proposal) => proposal !== undefined),
    [all, request.proposalIds],
  );
  const planId =
    request.planId !== "" ? request.planId : requested.length === 1 ? requested[0].planId : "";
  const plansQuery = useQuery({ ...queries.assistant.plans(threadId), enabled: planId !== "" });
  const plan = useMemo(
    () => plansQuery.data?.results.find((candidate) => candidate.id === planId) ?? null,
    [plansQuery.data, planId],
  );
  const steps = useMemo(
    () =>
      all
        .filter((candidate) => planId !== "" && candidate.planId === planId)
        .sort((a, b) => a.planStep - b.planStep),
    [all, planId],
  );

  if (proposalsQuery.isPending || (planId !== "" && plansQuery.isPending)) {
    return <Skeleton className="h-24 w-full" aria-label={t("Loading the proposal")} />;
  }

  if (plan !== null) {
    return <PlanCard plan={plan} steps={steps} threadId={threadId} />;
  }

  if (requested.length === 0 || request.planId !== "") {
    return (
      <p className="text-muted-foreground text-xs">
        {t("This proposal is no longer part of the conversation.")}
      </p>
    );
  }

  if (requested.length === 1) {
    return <ProposalCard proposal={requested[0]} threadId={threadId} />;
  }

  return (
    <div className="flex flex-col gap-2">
      <ProposalBatchBar proposals={requested} threadId={threadId} />
      {requested.map((proposal) => (
        <ProposalCard key={proposal.id} proposal={proposal} threadId={threadId} />
      ))}
    </div>
  );
}
