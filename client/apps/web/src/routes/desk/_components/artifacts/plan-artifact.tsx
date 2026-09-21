import { PlanCard } from "@/components/assistant/plan-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useMemo } from "react";
import { planFrom } from "./artifact-payloads";

/**
 * A plan as a checklist. The card is the plan's own, read from the plan and
 * its steps, so a step that ran ticks here the moment it ran.
 */
export function PlanArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const summary = useMemo(() => planFrom(artifact), [artifact]);
  const plansQuery = useQuery(queries.assistant.plans(artifact.threadId));
  const proposalsQuery = useQuery(queries.assistant.proposals(artifact.threadId));

  const plan = plansQuery.data?.results.find((candidate) => candidate.id === artifact.planId) ?? null;
  const steps = useMemo(
    () =>
      (proposalsQuery.data?.results ?? [])
        .filter((proposal) => proposal.planId === artifact.planId)
        .sort((a, b) => a.planStep - b.planStep),
    [artifact.planId, proposalsQuery.data?.results],
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
      {summary.summary !== "" && (
        <p className="text-muted-foreground text-sm leading-relaxed">{summary.summary}</p>
      )}
      {plansQuery.isLoading || proposalsQuery.isLoading ? (
        <Skeleton className="h-32" />
      ) : plan ? (
        <PlanCard plan={plan} steps={steps} threadId={artifact.threadId} />
      ) : (
        <ol className="flex flex-col gap-1.5 text-sm">
          {summary.steps.map((step) => (
            <li key={step.proposalId || step.step} className="flex gap-2">
              <span className="text-muted-foreground w-5 shrink-0 text-right tabular-nums">
                {step.step}.
              </span>
              <span className="min-w-0 flex-1">
                <span className="font-medium">{step.toolName}</span>
                {step.rationale !== "" && (
                  <span className="text-muted-foreground block text-xs">{step.rationale}</span>
                )}
              </span>
            </li>
          ))}
          {summary.steps.length === 0 && (
            <li className="text-muted-foreground text-xs">
              {t("The plan behind this is no longer in the conversation.")}
            </li>
          )}
        </ol>
      )}
    </div>
  );
}
