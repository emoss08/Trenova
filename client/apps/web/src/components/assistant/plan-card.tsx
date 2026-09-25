import { Button } from "@trenova/shared/components/ui/button";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { decideMyPlan } from "@/lib/graphql/agent-decisions";
import type {
  PlanPreview,
  ProposalPreview as ProposalPreviewData,
} from "@/lib/graphql/agent-preview";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import type { AssistantPlan, AssistantProposal, PlanDecision } from "@/types/assistant";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
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
import { StepDependencyNote } from "./proposal-preview/plan-preview";
import { canApprove, gateDigest } from "./proposal-preview/preview-gate";
import {
  PreviewLoadState,
  ProposalPreview,
  StaleNotice,
} from "./proposal-preview/proposal-preview";
import { useApprovalGate, usePlanPreview } from "./proposal-preview/use-proposal-preview";
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

  const awaiting = state === "awaiting";
  const undecided = awaiting || state === "held";
  // Every pending step previewed in order, each starting from what the steps
  // before it leave. One digest covers them all and goes with the approval.
  const previewQuery = usePlanPreview({ scope: "mine", id: plan.id, enabled: undecided });
  const approval = useApprovalGate(previewQuery);
  const previews = useMemo(() => previewsByStep(previewQuery.data), [previewQuery.data]);

  // Decided as the person whose conversation raised it, which needs only the
  // assistant, the way a single proposal in the thread is. Approving still
  // runs each step as them, so a step they may not make fails on its own.
  const decideMutation = useMutation({
    mutationFn: ({ decision, previewDigest }: { decision: PlanDecision; previewDigest?: string }) =>
      decideMyPlan(plan.id, { decision, reasonCode: "", previewDigest }),
    onSuccess: () => invalidateProposalViews(queryClient, threadId),
    onError: (error) => {
      // A digest that no longer matches: a step would now do something else.
      // The plan is read again and the card says so, rather than an error.
      if (!approval.handleDecisionError(error)) {
        handleMutationError({ error, resourceName: "Plan" });
      }
      // Any other refusal usually means the plan was decided elsewhere;
      // catching up beats a card stuck on a question nobody can answer.
      void invalidateProposalViews(queryClient, threadId);
    },
  });
  const decide = (decision: PlanDecision) => {
    approval.acknowledge();
    decideMutation.mutate({ decision, previewDigest: gateDigest(approval.gate) });
  };

  const decidedHere = useWatchedChange(undecided);
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
              onClick={() => decide("Accepted")}
              disabled={decideMutation.isPending || !canApprove(approval.gate)}
              isLoading={
                decideMutation.isPending && decideMutation.variables?.decision === "Accepted"
              }
            >
              <CheckIcon className="size-3.5" />
              {t("Approve all")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => decide("Rejected")}
              disabled={decideMutation.isPending}
              isLoading={
                decideMutation.isPending && decideMutation.variables?.decision === "Rejected"
              }
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

      {steps.length > 0 && <StepList steps={steps} settled={false} previews={previews} />}

      <PreviewLoadState query={previewQuery} changed={approval.changed} density="compact">
        {(preview) =>
          preview.stale && !preview.steps.some((step) => step.preview.stale) ? (
            <StaleNotice missing={false} />
          ) : null
        }
      </PreviewLoadState>
    </DecisionFrame>
  );
}

/** Each pending step's preview, by the proposal it belongs to. */
function previewsByStep(
  preview: PlanPreview | undefined,
): ReadonlyMap<string, ProposalPreviewData> {
  return new Map((preview?.steps ?? []).map((step) => [step.proposalId, step.preview]));
}

/**
 * The steps in the order they run. Before a decision each is the sentence
 * the proposal card would show with what it would change beneath; after one,
 * each also says what became of it, so a plan that stopped shows exactly
 * where.
 */
function StepList({
  steps,
  settled,
  previews,
}: {
  steps: AssistantProposal[];
  settled: boolean;
  previews?: ReadonlyMap<string, ProposalPreviewData>;
}) {
  return (
    <ol className={cn("flex flex-col gap-1.5 text-xs", settled && "pl-8.5", previews && "gap-3")}>
      {steps.map((step) => (
        <PlanStep key={step.id} step={step} settled={settled} preview={previews?.get(step.id)} />
      ))}
    </ol>
  );
}

function PlanStep({
  step,
  settled,
  preview,
}: {
  step: AssistantProposal;
  settled: boolean;
  preview?: ProposalPreviewData;
}) {
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
      <div className="min-w-0 flex-1">
        <span
          className={cn("block", stepState === "skipped" && "text-foreground-subtle line-through")}
        >
          {view.summary}
        </span>
        {!settled && preview && (
          <div className="mt-1.5 flex flex-col gap-2">
            <StepDependencyNote preview={preview} />
            <ProposalPreview preview={preview} density="compact" inPlan />
          </div>
        )}
        {settled && stepState === "failed" && step.executionError !== "" && (
          <span className="text-danger block">{step.executionError}</span>
        )}
        {settled && stepState === "simulated" && step.simulation?.summary && (
          <span className="text-foreground-muted block">{step.simulation.summary}</span>
        )}
      </div>
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
