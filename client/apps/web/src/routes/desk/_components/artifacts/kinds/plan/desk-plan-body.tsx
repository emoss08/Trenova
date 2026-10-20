import {
  canApprove,
  gateDigest,
  reviewRequiredFor,
} from "@/components/assistant/proposal-preview/preview-gate";
import {
  ChangedNotice,
  StaleNotice,
} from "@/components/assistant/proposal-preview/proposal-preview";
import {
  useApprovalGate,
  usePlanPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { holdReasonText, holdSwitchOf } from "@/components/assistant/approval-queue";
import { ReasonDialog } from "@/components/reason-dialog";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { decideMyPlan, undoMyDecision } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews, markPlanDecided, markPlanStatus } from "@/lib/proposal-cache";
import { assistant } from "@/lib/queries/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type {
  AssistantArtifact,
  AssistantPlan,
  AssistantProposal,
  AssistantThread,
} from "@/types/assistant";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMemo, useState } from "react";
import { planFrom, type PlanArtifact } from "../../artifact-payloads";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardSection,
  CardStatus,
  CardTimeline,
  RecordCard,
  type CardTone,
} from "../../record-card/record-card";
import { cardMoment } from "../../record-card/record-card-values";
import {
  countdown,
  finishMessage,
  foldedByDefault,
  phaseLabel,
  planLog,
  planPhase,
  planSteps,
  recordsTouched,
  runModeLine,
  shortDigest,
  stepsRun,
  supersededBy,
  undecided,
  unfinished,
  type PlanPhase,
  type PlanStepView,
} from "./plan-card-model";
import { logMoment, PlanNote, PlanRecords, PlanStep, PlanTrack } from "./plan-parts";
import { PlanSkeleton } from "./plan-skeleton";

/** The longest note a decision keeps, as the server counts it. */
const DECISION_NOTE_LENGTH = 2000;

const PHASE_TONES: Record<PlanPhase, CardTone> = {
  awaiting: "warn",
  held: "warn",
  undoable: "progress",
  running: "progress",
  failed: "danger",
  done: "success",
  declined: "muted",
  expired: "muted",
};

/** The plan a plan artifact stands for: the one it names, else the one its steps belong to. */
function planIdOf(
  artifact: AssistantArtifact,
  payload: PlanArtifact,
  proposals: AssistantProposal[],
) {
  if (artifact.planId !== "") {
    return artifact.planId;
  }
  const stepIds = new Set(payload.steps.map((step) => step.proposalId));
  return (
    proposals.find((proposal) => stepIds.has(proposal.id) && proposal.planId !== "")?.planId ?? ""
  );
}

/**
 * A plan: its steps as a track, what each changes from what to what, the
 * records it touches and how it was decided. While it waits it is approved or
 * declined here, against the preview drawn on it; once it runs, each step
 * says how it went and folds to a line.
 */
export function DeskPlanBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const payload = useMemo(() => planFrom(artifact), [artifact]);
  const proposalsQuery = useQuery(assistant.proposals(artifact.threadId));
  const plansQuery = useQuery(assistant.plans(artifact.threadId));
  const threadQuery = useQuery(assistant.thread(artifact.threadId));

  if (proposalsQuery.isPending || plansQuery.isPending) {
    return <PlanSkeleton />;
  }

  const proposals = proposalsQuery.data?.results ?? [];
  const plans = plansQuery.data?.results ?? [];
  const planId = planIdOf(artifact, payload, proposals);
  const plan = plans.find((candidate) => candidate.id === planId) ?? null;

  if (plan === null) {
    return (
      <RecordCard className="dk-pln">
        <CardHead
          kicker={[t("Plan")]}
          title={payload.title}
          subtitle={t("The plan behind this is no longer in the conversation.")}
        />
      </RecordCard>
    );
  }

  return (
    <PlanCard
      plan={plan}
      plans={plans}
      payload={payload}
      steps={proposals.filter((proposal) => proposal.planId === plan.id)}
      thread={threadQuery.data ?? null}
      threadId={artifact.threadId}
    />
  );
}

function PlanCard({
  plan,
  plans,
  payload,
  steps,
  thread,
  threadId,
}: {
  plan: AssistantPlan;
  plans: AssistantPlan[];
  payload: PlanArtifact;
  steps: AssistantProposal[];
  thread: AssistantThread | null;
  threadId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const viewerId = useAuthStore((state) => state.user?.id ?? "");
  const askAgent = useDeskStore((state) => state.askAgent);
  const asked = useDeskStore((state) => state.asks[threadId] ?? null);
  const now = useNowSeconds(plan.status === "Approving" ? 1000 : 30_000);
  const previewQuery = usePlanPreview({ scope: "mine", id: plan.id });
  const approval = useApprovalGate(previewQuery);
  const [declining, setDeclining] = useState(false);

  const phase = planPhase(plan, now);
  const views = planSteps(payload, steps, previewQuery.data, phase);
  const { records, withheld } = recordsTouched(views);
  const ownsThread = thread !== null && viewerId !== "" && thread.userId === viewerId;
  const canAct = ownsThread && thread.canContinue;
  const canDecide = canAct && phase === "awaiting";
  const canUndo = canAct && phase === "undoable" && plan.decidedByUserId === viewerId;
  const later = supersededBy(plan, plans);
  const canFinish = canAct && unfinished(phase, views) && later === null;
  const reviewRequired = reviewRequiredFor(
    previewQuery.data,
    steps.length === 0 || steps.some((step) => step.reviewRequired),
    false,
  );

  const settle = () => void invalidateProposalViews(queryClient, threadId);
  const approve = useMutation({
    mutationFn: () =>
      decideMyPlan(plan.id, {
        decision: "Accepted",
        reasonCode: "approved_from_plan_card",
        previewDigest: gateDigest(approval.gate),
      }),
    onSuccess: (decided) => {
      if ((decided.commitsAt ?? 0) > 0) {
        markPlanStatus(queryClient, plan.id, "Approving");
      } else {
        markPlanDecided(queryClient, plan.id, "Accepted");
      }
      settle();
    },
    onError: (error) => {
      if (!approval.handleDecisionError(error)) {
        handleMutationError({ error, resourceName: "Plan" });
      }
      settle();
    },
  });
  const decline = useMutation({
    mutationFn: (note: string) =>
      decideMyPlan(plan.id, {
        decision: "Rejected",
        reasonCode: "declined_from_plan_card",
        note: note === "" ? undefined : note,
      }),
    onSuccess: () => {
      setDeclining(false);
      markPlanDecided(queryClient, plan.id, "Rejected");
      settle();
    },
    onError: (error) => {
      handleMutationError({ error, resourceName: "Plan" });
      settle();
    },
  });
  const undo = useMutation({
    mutationFn: () => undoMyDecision({ planId: plan.id }),
    onSuccess: () => {
      markPlanStatus(queryClient, plan.id, "Pending");
      settle();
    },
    onError: (error) => {
      handleMutationError({ error, resourceName: "Plan" });
      settle();
    },
  });

  const busy = approve.isPending || decline.isPending || undo.isPending;
  const approvable = canDecide && canApprove(approval.gate, reviewRequired) && !busy;
  const total = Math.max(views.length, plan.stepCount, 1);
  const agent = plan.agentName?.trim() || t("The agent");
  const decider = plan.decidedByUserId === viewerId ? t("you") : plan.decidedByName || t("someone");
  const firstMoment = plan.createdAt;
  const left = Math.max(0, (plan.commitsAt ?? 0) - now);
  const stale = previewQuery.data?.stale === true;
  const held = holdSwitchOf(plan.hold, plan);

  const figure = undecided(phase) ? (
    <>
      <b>{t("{0, plural, one {# change} other {# changes}}", total)}</b>
      <span>
        {records.length > 0
          ? t("{0, plural, one {on # record} other {on # records}}", records.length)
          : t("to approve")}
      </span>
    </>
  ) : (
    <>
      <b>{t("{0} of {1}", stepsRun(views), total)}</b>
      <span>{t("steps run")}</span>
    </>
  );

  const facts = [
    t("Proposed by {0}", agent),
    cardMoment(plan.createdAt),
    undecided(phase)
      ? (plan.expiresAt ?? 0) > 0
        ? t("Expires {0}", cardMoment(plan.expiresAt))
        : ""
      : phase === "expired"
        ? t("Nobody decided it")
        : plan.status === "Rejected"
          ? t("Declined by {0}", decider)
          : t("Approved by {0}", decider),
  ];

  const log = planLog(plan, views, phase, viewerId, t).map((entry) => ({
    id: entry.id,
    text: entry.text,
    mine: entry.mine,
    meta: logMoment(firstMoment, entry.at),
    detail: entry.note !== "" ? <PlanNote>{entry.note}</PlanNote> : undefined,
  }));

  const failed = views.filter((step) => step.tone === "fail");
  const dependencyLabel = (numbers: readonly number[]) =>
    numbers.length === 1
      ? t("Starts from what step {0} leaves", numbers[0])
      : t("Starts from what steps {0} leave", numbers.join(", "));

  return (
    <RecordCard className="dk-pln">
      <CardHead
        kicker={[
          t("Plan"),
          t("{0, plural, one {# step} other {# steps}}", total),
          plan.runMode === "EachStep" ? t("Every step runs") : t("Runs in order"),
        ]}
        title={plan.title || payload.title}
        subtitle={plan.summary || payload.summary || undefined}
        figure={figure}
        under={<PlanTrack tones={views.map((step) => step.tone)} />}
        meta={
          <>
            <CardStatus
              tone={PHASE_TONES[phase]}
              mark={phase === "done" ? "check" : phase === "held" ? "pause" : "dot"}
            >
              {phaseLabel(phase, views, plan.runMode, canDecide, t)}
            </CardStatus>
            <CardFacts>{facts}</CardFacts>
          </>
        }
      />
      {phase === "held" && held !== null && (
        <section className="dk-rec-sec">
          <CardChecks>
            <CardCheck
              state="warn"
              title={t("This plan is on hold")}
              detail={holdReasonText(held, t)}
            />
          </CardChecks>
        </section>
      )}
      {phase === "failed" && failed.length > 0 && (
        <section className="dk-rec-sec">
          <CardChecks>
            <CardCheck
              state="bad"
              title={t("{0} of {1} steps went through", stepsRun(views), total)}
              detail={
                plan.runMode === "EachStep"
                  ? plan.failureError || t("The other steps still ran.")
                  : t(
                      "Step {0} stopped the plan, so the steps after it did not run.",
                      failed[0]?.number ?? plan.failedStep ?? 0,
                    )
              }
            />
          </CardChecks>
        </section>
      )}
      {canDecide && (approval.changed || stale) && (
        <section className="dk-rec-sec">
          {stale ? <StaleNotice missing={false} /> : <ChangedNotice />}
        </section>
      )}
      <CardSection
        title={t("Steps")}
        aside={<span>{runModeLine(phase, plan.runMode, views, t)}</span>}
      >
        <ol className="dk-pln-steps">
          {views.map((step: PlanStepView) => (
            <PlanStep
              key={step.proposalId || step.number}
              step={step}
              folded={foldedByDefault(step, phase)}
              dependencyLabel={dependencyLabel}
            />
          ))}
        </ol>
      </CardSection>
      {(records.length > 0 || withheld > 0) && (
        <CardSection
          title={t("Records it touches")}
          aside={<span>{t("{0, plural, one {# record} other {# records}}", records.length)}</span>}
        >
          <PlanRecords records={records} withheld={withheld} />
        </CardSection>
      )}
      <CardSection
        title={t("Approval")}
        aside={
          plan.previewDigest ? (
            <span className="dk-pln-dig">
              {t("Approved against {0}", shortDigest(plan.previewDigest))}
            </span>
          ) : previewQuery.data && undecided(phase) ? (
            <span className="dk-pln-dig">
              {t("Preview {0}", shortDigest(previewQuery.data.digest))}
            </span>
          ) : undefined
        }
      >
        <CardTimeline entries={log} />
      </CardSection>
      <PlanActions
        phase={phase}
        canDecide={canDecide}
        canUndo={canUndo}
        canFinish={canFinish}
        superseded={later !== null}
        approvable={approvable}
        busy={busy}
        asked={asked !== null}
        secondsLeft={left}
        ran={stepsRun(views)}
        total={total}
        onApprove={() => {
          approval.acknowledge();
          approve.mutate();
        }}
        onDecline={() => setDeclining(true)}
        onUndo={() => undo.mutate()}
        onFinish={() => askAgent(threadId, finishMessage(plan, views, phase, t))}
      />
      <ReasonDialog
        open={declining}
        title={t("Decline this plan?")}
        description={t(
          "None of its steps will run. Anything you write here goes to the agent with the decision.",
        )}
        label={t("Note for the agent")}
        placeholder={t("Optional")}
        confirmLabel={t("Decline plan")}
        destructive
        optional
        maxLength={DECISION_NOTE_LENGTH}
        pending={decline.isPending}
        onConfirm={(note) => decline.mutate(note)}
        onOpenChange={setDeclining}
      />
    </RecordCard>
  );
}

/** The card's bar: what the person can do with the plan as it stands, and why not when they cannot. */
function PlanActions({
  phase,
  canDecide,
  canUndo,
  canFinish,
  superseded,
  approvable,
  busy,
  asked,
  secondsLeft,
  ran,
  total,
  onApprove,
  onDecline,
  onUndo,
  onFinish,
}: {
  phase: PlanPhase;
  canDecide: boolean;
  canUndo: boolean;
  canFinish: boolean;
  superseded: boolean;
  approvable: boolean;
  busy: boolean;
  asked: boolean;
  secondsLeft: number;
  ran: number;
  total: number;
  onApprove: () => void;
  onDecline: () => void;
  onUndo: () => void;
  onFinish: () => void;
}) {
  const t = useT();

  if (canDecide) {
    return (
      <CardBar>
        <CardBarHint>{t("You approve exactly what is shown here.")}</CardBarHint>
        <Button variant="outline" disabled={busy} onClick={onDecline}>
          {t("Decline")}
        </Button>
        <Button disabled={!approvable} isLoading={busy} onClick={onApprove}>
          {t("Approve plan")}
        </Button>
      </CardBar>
    );
  }

  if (phase === "undoable") {
    return (
      <CardBar>
        <CardBarHint>
          <span className="dk-pln-undo" role="timer" aria-live="off">
            {t("Can be undone for {0}", countdown(secondsLeft))}
          </span>
        </CardBarHint>
        {canUndo && (
          <Button variant="outline" disabled={busy} isLoading={busy} onClick={onUndo}>
            {t("Undo")}
          </Button>
        )}
      </CardBar>
    );
  }

  if (canFinish) {
    return (
      <CardBar>
        <CardBarHint>{t("{0} of {1} changes are in place.", ran, total)}</CardBarHint>
        <Button disabled={asked} onClick={onFinish}>
          {t("Ask the agent to finish it")}
        </Button>
      </CardBar>
    );
  }

  const hint = superseded
    ? t("A later plan in this conversation takes it from here.")
    : phase === "awaiting" || phase === "held"
      ? t("It is decided in the conversation it was proposed in.")
      : phase === "running"
        ? t("The steps left run on their own; nothing here needs you.")
        : phase === "done"
          ? t("Ran as approved.")
          : phase === "declined"
            ? t("Nothing changed. The agent was told.")
            : phase === "expired"
              ? t("Nothing changed.")
              : t("{0} of {1} changes are in place.", ran, total);

  return (
    <CardBar>
      <CardBarHint>{hint}</CardBarHint>
    </CardBar>
  );
}
