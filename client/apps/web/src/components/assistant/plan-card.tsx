import { Button } from "@trenova/shared/components/ui/button";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { toneVar } from "@/components/kpi/tone";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantPlan, AssistantProposal, PlanDecision } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  ListChecksIcon,
  LoaderIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import { m } from "motion/react";
import {
  classifyPlan,
  planStepState,
  type PlanPresentation,
  type PlanStepState,
} from "./plan-state";
import { HoldLine, OutcomeIcon } from "./proposal-card";
import { presentProposal } from "./proposal-presenters";

/**
 * Several writes the assistant is asking for as one.
 *
 * A run that needs three changes made in order is one question, not three:
 * approving the second without the first would leave the work half done. So
 * the card lists the steps in the order they will run and takes one answer
 * for all of them. Once approved it keeps following the steps, because
 * "approved" and "done" are different facts and a step that failed stops the
 * ones after it.
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
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.assistant.plans(threadId).queryKey }),
        queryClient.invalidateQueries({
          queryKey: queries.assistant.proposals(threadId).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.assistant.messages(threadId).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ["assistant", "pending-proposals"] }),
      ]);
    },
    resourceName: "Plan",
  });

  const awaiting = state === "awaiting";
  const permanent = steps.some((step) => !presentProposal(step).reversible);

  if (!awaiting && state !== "held") {
    return (
      <m.div
        layout
        className="border-border/70 text-muted-foreground flex flex-col gap-1.5 rounded-lg border px-3 py-2 text-xs"
      >
        <div className="flex items-start gap-2">
          <OutcomeIcon state={state} />
          <span className="min-w-0 flex-1">
            <span className="text-foreground block">{plan.title}</span>
            <PlanOutcomeLine plan={plan} state={state} />
          </span>
        </div>
        {steps.length > 0 && <StepList steps={steps} settled />}
      </m.div>
    );
  }

  return (
    <m.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-card ring-foreground/10 flex flex-col overflow-hidden rounded-xl ring-1"
    >
      <div className="flex flex-col gap-2 px-3.5 pt-3 pb-2.5">
        <div className="flex items-center gap-2">
          <span
            aria-hidden
            className="size-1.5 shrink-0 rounded-full"
            style={{ backgroundColor: toneVar("warning") }}
          />
          <span className="text-muted-foreground flex min-w-0 flex-1 items-center gap-1.5 truncate text-xs">
            <ListChecksIcon className="size-3.5 shrink-0" />
            {t("{0, plural, one {# change, in order} other {# changes, in order}}", plan.stepCount)}
          </span>
        </div>

        <p className="text-sm leading-snug">{plan.title}</p>

        {plan.summary !== "" && (
          <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-wrap">
            {plan.summary}
          </p>
        )}

        {steps.length > 0 && <StepList steps={steps} settled={false} />}
      </div>

      {state === "held" ? (
        <HoldLine hold={plan.hold} />
      ) : (
        <div className="flex items-center gap-2 px-3 pb-3">
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

          <span className="ml-auto flex items-center gap-2">
            {permanent && (
              <span
                className="flex items-center gap-1 text-xs"
                style={{ color: toneVar("warning") }}
              >
                <TriangleAlertIcon className="size-3" />
                {t("Permanent")}
              </span>
            )}
          </span>
        </div>
      )}
    </m.div>
  );
}

/**
 * The steps in the order they run. Before a decision each is the sentence
 * the proposal card would show; after one, each also says what became of it,
 * so a plan that stopped shows exactly where.
 */
function StepList({ steps, settled }: { steps: AssistantProposal[]; settled: boolean }) {
  return (
    <ol className={cn("flex flex-col gap-1.5", settled ? "pl-5.5" : "text-xs")}>
      {steps.map((step) => {
        const view = presentProposal(step);
        const stepState = planStepState(step);

        return (
          <li key={step.id} className="flex items-start gap-2">
            {settled ? (
              <StepIcon state={stepState} />
            ) : (
              <span className="text-muted-foreground w-4 shrink-0 text-right tabular-nums">
                {step.planStep}.
              </span>
            )}
            <span className="min-w-0 flex-1">
              <span className={cn("block", stepState === "skipped" && "line-through")}>
                {view.summary}
              </span>
              {settled && stepState === "failed" && step.executionError !== "" && (
                <span className="block" style={{ color: toneVar("danger") }}>
                  {step.executionError}
                </span>
              )}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

function StepIcon({ state }: { state: PlanStepState }) {
  const className = "mt-px size-3.5 shrink-0";

  switch (state) {
    case "failed":
      return <CircleAlertIcon className={className} style={{ color: toneVar("danger") }} />;
    case "done":
      return <CircleCheckIcon className={className} style={{ color: toneVar("success") }} />;
    case "running":
      return <LoaderIcon className={cn(className, "animate-spin")} />;
    default:
      return <CircleSlashIcon className={cn(className, "text-muted-foreground")} />;
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
        <span className="block" style={{ color: toneVar("danger") }}>
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
