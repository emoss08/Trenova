import type { PlanPreview as PlanPreviewData, ProposalPreview } from "@/lib/graphql/agent-preview";
import { useT } from "@trenova/shared/i18n/use-t";
import { LinkIcon } from "lucide-react";
import type { ReactNode } from "react";
import { dependencySteps } from "./preview-format";
import {
  ProposalPreview as ProposalPreviewView,
  StaleNotice,
  type PreviewDensity,
  type WouldFailActions,
} from "./proposal-preview";

/**
 * "Uses the record step 1 changes": a step that starts from what an earlier
 * step leaves, said once per earlier step, so a reader knows its "before" is
 * a projection rather than the record as it is.
 */
export function StepDependencyNote({ preview }: { preview: ProposalPreview }) {
  const t = useT();
  const steps = dependencySteps(preview);
  if (steps.length === 0) {
    return null;
  }

  return (
    <span className="text-foreground-muted flex flex-col gap-0.5 text-xs">
      {steps.map((step) => (
        <span key={step} className="inline-flex items-center gap-1">
          <LinkIcon aria-hidden className="size-3 shrink-0" />
          {t("Uses the record step {0} changes", step)}
        </span>
      ))}
    </span>
  );
}

/**
 * What every pending step of a plan would do, in the order the steps run. A
 * step on a record an earlier step changes is shown as that step would leave
 * it, and says so. If any step's record moved since the plan was proposed the
 * whole plan can only be rejected, and the step that moved says why.
 */
export function PlanPreview({
  plan,
  density = "full",
  stepTitle,
  wouldFail,
}: {
  plan: PlanPreviewData;
  density?: PreviewDensity;
  /** The sentence a surface already has for a step; "Step 2" when it has none. */
  stepTitle?: (proposalId: string, step: number) => ReactNode;
  /** What the surface offers when a step would be refused; a plan's steps cannot be edited one by one. */
  wouldFail?: WouldFailActions;
}) {
  const t = useT();
  const anyStepStale = plan.steps.some((step) => step.preview.stale);

  return (
    <div className="flex min-w-0 flex-col gap-3" data-slot="plan-preview">
      {plan.stale && !anyStepStale && <StaleNotice missing={false} />}
      {plan.steps.length === 0 ? (
        <p className="text-foreground-muted text-xs">{t("No steps are left to run.")}</p>
      ) : (
        <ol className="flex flex-col gap-4">
          {plan.steps.map((step) => (
            <li key={step.proposalId} className="flex min-w-0 gap-2">
              <span className="text-foreground-subtle w-5 shrink-0 text-right text-sm tabular-nums">
                {step.step}.
              </span>
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                <span className="text-sm">
                  {stepTitle ? stepTitle(step.proposalId, step.step) : t("Step {0}", step.step)}
                </span>
                <StepDependencyNote preview={step.preview} />
                <ProposalPreviewView
                  preview={step.preview}
                  density={density}
                  inPlan
                  wouldFail={wouldFail}
                />
              </div>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
