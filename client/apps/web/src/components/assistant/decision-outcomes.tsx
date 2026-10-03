import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  PlanPreview,
  ProposalPreview as ProposalPreviewData,
} from "@/lib/graphql/agent-preview";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import {
  AlertCircleIcon,
  Beaker02Icon,
  CheckCircleIcon,
  PauseCircleIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { useWatchedChange } from "./decision-chrome";
import { planStepState, type PlanPresentation, type PlanStepState } from "./plan-state";
import { StepDependencyNote } from "./proposal-preview/plan-preview";
import { ProposalPreview, type WouldFailActions } from "./proposal-preview/proposal-preview";
import { presentProposal } from "./proposal-presenters";
import { ranWithoutApproval, type ProposalPresentation } from "./proposal-state";
import { WorkingDot } from "./voice/working-dot";

/**
 * What the approver changed before approving, so a settled decision says
 * what ran rather than only that something did.
 */
export function ChangesLine({
  modifications,
}: {
  modifications: AssistantProposal["modifications"];
}) {
  const t = useT();
  const entries = Object.entries(modifications ?? {});
  if (entries.length === 0) {
    return null;
  }

  return (
    <span className="block">
      {t("Approved with changes:")}{" "}
      {entries.map(([key, value], index) => (
        <span key={key}>
          {index > 0 ? ", " : ""}
          {key}: {formatChange(value)}
        </span>
      ))}
    </span>
  );
}

function formatChange(value: unknown): string {
  if (value === null || value === undefined) return "—";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

/**
 * What the person told the agent with the decision, in their own words, so
 * the record of a "Tell the agent" says what was said.
 */
export function DecisionNoteLine({ note }: { note: string }) {
  const t = useT();
  if (note.trim() === "") {
    return null;
  }

  return (
    <span className="block whitespace-pre-wrap">{t("You told the agent: “{0}”", note.trim())}</span>
  );
}

/**
 * Why nobody can decide yet, naming the switch.
 *
 * There are two switches on two tabs, and the organization-wide pause is on
 * from the day an organization is created. A line that said only "shadow mode"
 * sent people to turn it off on the agent, where it already was, and back to
 * the same refusal.
 */
export function HoldLine({ hold }: { hold: AssistantProposal["hold"] }) {
  const t = useT();
  if (!hold) {
    return null;
  }

  return (
    <p className="text-foreground-muted flex items-center gap-1.5 text-xs">
      <PauseCircleIcon className="size-3.5 shrink-0" />
      <span>
        {hold.reason === "AgentShadow" && hold.agentName !== ""
          ? t("On hold: {0} is in shadow mode in AI Control.", hold.agentName)
          : t("On hold: all agents are paused in AI Control.")}
      </span>
    </p>
  );
}

/**
 * What a simulated write would have changed. The approver cleared it and it
 * did not happen, on purpose, so the line says both and shows the preview
 * rather than a success.
 */
export function SimulationLine({ simulation }: { simulation: AssistantProposal["simulation"] }) {
  const t = useT();

  return (
    <span className="block">
      <span className="block">{t("Simulated: approved, and nothing was changed.")}</span>
      {simulation?.summary ? <span className="block">{simulation.summary}</span> : null}
      {simulation && simulation.changes.length > 0 && (
        <ul className="mt-1 flex flex-col gap-0.5">
          {simulation.changes.map((change) => (
            <li key={change.field} className="tabular-nums">
              {change.field}: {change.from !== "" ? `${change.from} → ` : ""}
              {change.to}
            </li>
          ))}
        </ul>
      )}
    </span>
  );
}

/**
 * What the presenters read out of the raw arguments: what can still be said
 * when the preview itself could not be read.
 */
export function Highlights({ highlights }: { highlights: { label: string; value: string }[] }) {
  if (highlights.length === 0) {
    return null;
  }

  return (
    <dl className="flex flex-col gap-1.5 text-xs">
      {highlights.map((entry) => (
        <HighlightRow key={entry.label} label={entry.label} value={entry.value} />
      ))}
    </dl>
  );
}

/**
 * A short value reads as a label/value pair; a sentence does not. Running
 * every value through a fixed gutter turned "what the agent found" into a
 * narrow column beside a wall of wrapped text.
 */
const INLINE_VALUE_LIMIT = 48;

function HighlightRow({ label, value }: { label: string; value: string }) {
  if (value.length > INLINE_VALUE_LIMIT) {
    return (
      <div>
        <dt className="text-foreground-subtle">{label}</dt>
        <dd className="mt-0.5 break-words">{value}</dd>
      </div>
    );
  }

  return (
    <div className="flex gap-3">
      <dt className="text-foreground-subtle w-24 shrink-0 truncate">{label}</dt>
      <dd className="min-w-0 flex-1 break-words">{value}</dd>
    </div>
  );
}

/**
 * What became of a proposal, stated separately from the decision itself.
 *
 * "Approved" and "done" are different facts and the record never runs them
 * together: an approval whose tool failed says so, with the reason, because
 * the approver is the one who needs to know their instruction did not take
 * effect.
 */
export function OutcomeLine({
  proposal,
  state,
}: {
  proposal: AssistantProposal;
  state: ProposalPresentation;
}) {
  const t = useT();
  const own = ranWithoutApproval(proposal);

  switch (state) {
    case "failed":
      return (
        <>
          <ChangesLine modifications={proposal.modifications} />
          <span className="text-danger block">
            {own
              ? proposal.executionError === ""
                ? t("It ran on its own but did not go through. Nothing was changed.")
                : t("It ran on its own but did not go through: {0}", proposal.executionError)
              : proposal.executionError === ""
                ? t("Approved, but it did not run. Nothing was changed.")
                : t("Approved, but it did not run: {0}", proposal.executionError)}
          </span>
        </>
      );
    case "done": {
      const when = proposal.executedAt
        ? generateDateTimeStringFromUnixTimestamp(proposal.executedAt)
        : "";
      return (
        <>
          <ChangesLine modifications={proposal.modifications} />
          <span className="block">
            {own
              ? when !== ""
                ? t("Done on its own {0}. No approval was needed.", when)
                : t("Done on its own. No approval was needed.")
              : when !== ""
                ? t("Done {0}", when)
                : t("Done")}
          </span>
        </>
      );
    }
    case "running":
      return (
        <>
          <ChangesLine modifications={proposal.modifications} />
          <span className="block">
            {own
              ? t("Running on its own. No approval is needed.")
              : t("Approved. Waiting for it to run.")}
          </span>
        </>
      );
    case "declined":
      return <span className="block">{t("Rejected. Nothing was changed.")}</span>;
    case "simulated":
      return <SimulationLine simulation={proposal.simulation} />;
    case "awaiting":
      return (
        <span className="block">{t("Waiting on your decision. Nothing has changed yet.")}</span>
      );
    case "held":
      return <HoldLine hold={proposal.hold} />;
    default:
      return <span className="block">{t("Expired without a decision.")}</span>;
  }
}

/** Each pending step's preview, by the proposal it belongs to. */
export function previewsByStep(
  preview: PlanPreview | undefined,
): ReadonlyMap<string, ProposalPreviewData> {
  return new Map((preview?.steps ?? []).map((step) => [step.proposalId, step.preview]));
}

/**
 * A plan's steps in the order they run. Before a decision each is the
 * sentence a proposal shows with what it would change beneath; after one,
 * each also says what became of it, so a plan that stopped shows exactly
 * where.
 */
export function StepList({
  steps,
  settled,
  previews,
  attention = true,
  wouldFail,
}: {
  steps: AssistantProposal[];
  settled: boolean;
  previews?: ReadonlyMap<string, ProposalPreviewData>;
  /** Whether each step's preview leads with its own warnings; off where the surface draws them. */
  attention?: boolean;
  wouldFail?: WouldFailActions;
}) {
  return (
    <ol className={cn("flex flex-col gap-1.5 text-xs", previews && "gap-3")}>
      {steps.map((step) => (
        <PlanStep
          key={step.id}
          step={step}
          settled={settled}
          preview={previews?.get(step.id)}
          attention={attention}
          wouldFail={wouldFail}
        />
      ))}
    </ol>
  );
}

function PlanStep({
  step,
  settled,
  preview,
  attention,
  wouldFail,
}: {
  step: AssistantProposal;
  settled: boolean;
  preview?: ProposalPreviewData;
  attention: boolean;
  wouldFail?: WouldFailActions;
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
            <ProposalPreview
              preview={preview}
              density="compact"
              inPlan
              attention={attention}
              wouldFail={wouldFail}
            />
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
      return <AlertCircleIcon key={state} className={cn(className, "text-danger")} />;
    case "done":
      return <CheckCircleIcon key={state} className={cn(className, "text-success")} />;
    case "running":
      return (
        <span className="flex size-3.5 shrink-0 items-center justify-center pt-px">
          <WorkingDot working still />
        </span>
      );
    case "simulated":
      return <Beaker02Icon key={state} className={cn(className, "text-foreground-muted")} />;
    default:
      return <SlashCircle01Icon key={state} className={cn(className, "text-foreground-subtle")} />;
  }
}

/**
 * Whether a failed plan went on past its first failure: steps that each change
 * a different record run whatever became of the others, so a step after the
 * one that failed still ran and none was skipped.
 */
function ranEveryStep(plan: AssistantPlan, steps: AssistantProposal[]): boolean {
  const failedAt = plan.failedStep ?? 0;
  if (failedAt === 0 || steps.some((step) => planStepState(step) === "skipped")) {
    return false;
  }

  return steps.some((step) => {
    const state = planStepState(step);
    return step.planStep > failedAt && (state === "done" || state === "failed");
  });
}

/**
 * What became of the plan, kept apart from the decision. A plan that stopped
 * says which step stopped it and how far it got, because the approver is the
 * one who has to finish what did not run.
 */
export function PlanOutcomeLine({
  plan,
  state,
  steps,
}: {
  plan: AssistantPlan;
  state: PlanPresentation;
  steps: AssistantProposal[];
}) {
  const t = useT();

  switch (state) {
    case "failed":
      if (ranEveryStep(plan, steps)) {
        return (
          <span className="text-danger block">
            {t(
              "Approved. {0} of {1} done; the others did not go through, and each says why.",
              plan.completedSteps,
              plan.stepCount,
            )}
          </span>
        );
      }
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
    case "awaiting":
      return (
        <span className="block">{t("Waiting on your decision. Nothing has changed yet.")}</span>
      );
    case "held":
      return <HoldLine hold={plan.hold} />;
    default:
      return <span className="block">{t("Expired without a decision.")}</span>;
  }
}
