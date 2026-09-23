import { Button } from "@trenova/shared/components/ui/button";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { apiService } from "@/services/api";
import type { AssistantPlan, AssistantProposal, PlanDecision } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  FlaskConicalIcon,
  ListChecksIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import { DecisionFrame, DecisionReceipt, ProposedBy, useWatchedChange } from "./decision-chrome";
import {
  classifyPlan,
  planStepState,
  type PlanPresentation,
  type PlanStepState,
} from "./plan-state";
import { HoldLine } from "./proposal-card";
import { presentProposal } from "./proposal-presenters";
import { WorkingDot } from "./voice/working-dot";

/**
 * Several writes the assistant is asking for as one.
 *
 * A run that needs three changes made in order is one question, not three:
 * approving the second without the first would leave the work half done. So
 * the card lists the steps in the order they will run and takes one answer
 * for all of them. Once approved it keeps following the steps, because
 * "approved" and "done" are different facts and a step that failed stops the
 * ones after it; each step's mark settles as it finishes.
 */
export function PlanCard({
  plan,
  steps,
  threadId,
}: {
  plan: AssistantPlan;
  steps: AssistantProposal[];
  threadId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const state = classifyPlan(plan);

  const decideMutation = useApiMutation({
    mutationFn: (decision: PlanDecision) =>
      apiService.assistantService.decidePlan(plan.id, decision),
    onSuccess: () => invalidateProposalViews(queryClient, threadId),
    resourceName: "Plan",
  });

  const awaiting = state === "awaiting";
  const decidedHere = useWatchedChange(awaiting || state === "held");
  const permanent = steps.some((step) => !presentProposal(step).reversible);
  const byline = <ProposedBy agentId={plan.agentId} agentName={plan.agentName} />;

  if (!awaiting && state !== "held") {
    return (
      <DecisionReceipt
        state={state}
        summary={plan.title}
        byline={byline}
        arrived={decidedHere}
        footer={steps.length > 0 ? <StepList steps={steps} settled /> : null}
      >
        <PlanOutcomeLine plan={plan} state={state} />
      </DecisionReceipt>
    );
  }

  return (
    <DecisionFrame
      icon={ListChecksIcon}
      title={plan.title}
      state={state}
      byline={byline}
      footer={
        state === "held" ? (
          <HoldLine hold={plan.hold} />
        ) : (
          <div className="border-border-subtle flex flex-wrap items-center gap-2 border-t px-3 py-2.5">
            <Button
              size="sm"
              onClick={() => decideMutation.mutate("Accepted")}
              disabled={decideMutation.isPending}
              isLoading={decideMutation.isPending && decideMutation.variables === "Accepted"}
            >
              <CheckIcon className="size-3.5" />
              {t("Approve all")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => decideMutation.mutate("Rejected")}
              disabled={decideMutation.isPending}
              isLoading={decideMutation.isPending && decideMutation.variables === "Rejected"}
            >
              <XIcon className="size-3.5" />
              {t("Reject all")}
            </Button>
            {permanent && (
              <span className="text-warning ml-auto flex items-center gap-1 text-xs">
                <TriangleAlertIcon className="size-3" />
                {t("Permanent")}
              </span>
            )}
          </div>
        )
      }
    >
      <span className="text-foreground-muted -mt-1 text-xs">
        {t("{0, plural, one {# change, in order} other {# changes, in order}}", plan.stepCount)}
      </span>

      {plan.summary !== "" && (
        <p className="text-foreground-muted text-xs leading-relaxed whitespace-pre-wrap">
          {plan.summary}
        </p>
      )}

      {steps.length > 0 && <StepList steps={steps} settled={false} />}
    </DecisionFrame>
  );
}

/**
 * The steps in the order they run. Before a decision each is the sentence
 * the proposal card would show; after one, each also says what became of it,
 * so a plan that stopped shows exactly where.
 */
function StepList({ steps, settled }: { steps: AssistantProposal[]; settled: boolean }) {
  return (
    <ol className={cn("flex flex-col gap-1.5 text-xs", settled && "pl-8.5")}>
      {steps.map((step) => (
        <PlanStep key={step.id} step={step} settled={settled} />
      ))}
    </ol>
  );
}

function PlanStep({ step, settled }: { step: AssistantProposal; settled: boolean }) {
  const view = presentProposal(step);
  const stepState = planStepState(step);

  return (
    <li className="flex items-start gap-2">
      {settled ? (
        <StepIcon state={stepState} />
      ) : (
        <span className="text-foreground-subtle w-4 shrink-0 text-right tabular-nums">
          {step.planStep}.
        </span>
      )}
      <span className="min-w-0 flex-1">
        <span
          className={cn("block", stepState === "skipped" && "text-foreground-subtle line-through")}
        >
          {view.summary}
        </span>
        {settled && stepState === "failed" && step.executionError !== "" && (
          <span className="text-danger block">{step.executionError}</span>
        )}
        {settled && stepState === "simulated" && step.simulation?.summary && (
          <span className="text-foreground-muted block">{step.simulation.summary}</span>
        )}
      </span>
    </li>
  );
}

/** A step's mark; one that finishes while watched settles with the spring. */
function StepIcon({ state }: { state: PlanStepState }) {
  const watched = useWatchedChange(state);
  const className = cn("mt-px size-3.5 shrink-0", watched && "animate-confirm");

  switch (state) {
    case "failed":
      return <CircleAlertIcon key={state} className={cn(className, "text-danger")} />;
    case "done":
      return <CircleCheckIcon key={state} className={cn(className, "text-success")} />;
    case "running":
      return (
        <span className="flex size-3.5 shrink-0 items-center justify-center pt-px">
          <WorkingDot working still />
        </span>
      );
    case "simulated":
      return <FlaskConicalIcon key={state} className={cn(className, "text-foreground-muted")} />;
    default:
      return <CircleSlashIcon key={state} className={cn(className, "text-foreground-subtle")} />;
  }
}

/**
 * What became of the plan, kept apart from the decision. A plan that stopped
 * says which step stopped it and how far it got, because the approver is the
 * one who has to finish what did not run.
 */
function PlanOutcomeLine({ plan, state }: { plan: AssistantPlan; state: PlanPresentation }) {
  const t = useT();

  switch (state) {
    case "failed":
      return (
        <span className="text-danger block">
          {plan.failedStep
            ? t(
                "Approved, but step {0} of {1} did not run and the rest were skipped.",
                plan.failedStep,
                plan.stepCount,
              )
            : t("Approved, but it did not finish.")}
        </span>
      );
    case "done":
      return (
        <span className="block">
          {plan.decidedAt
            ? t(
                "All {0} done. Approved {1}",
                plan.stepCount,
                generateDateTimeStringFromUnixTimestamp(plan.decidedAt),
              )
            : t("All {0} done.", plan.stepCount)}
        </span>
      );
    case "running":
      return (
        <span className="block">
          {t("Approved. {0} of {1} done so far.", plan.completedSteps, plan.stepCount)}
        </span>
      );
    case "declined":
      return <span className="block">{t("Rejected. Nothing was changed.")}</span>;
    default:
      return <span className="block">{t("Expired without a decision.")}</span>;
  }
}
