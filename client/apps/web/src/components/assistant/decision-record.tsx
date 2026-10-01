import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { queries } from "@/lib/queries";
import { useAssistantStore, type DecisionFocus } from "@/stores/assistant-store";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { ChevronRightIcon, HourglassIcon, type LucideIcon } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { focusKeys } from "./approval-queue";
import { OutcomeIcon, ProposedBy } from "./decision-chrome";
import {
  DecisionNoteLine,
  Highlights,
  OutcomeLine,
  PlanOutcomeLine,
  StepList,
  previewsByStep,
} from "./decision-outcomes";
import { recordStatus } from "./decision-record-status";
import { classifyPlan } from "./plan-state";
import {
  PreviewLoadState,
  ProposalPreview,
  StaleNotice,
} from "./proposal-preview/proposal-preview";
import { usePlanPreview, useProposalPreview } from "./proposal-preview/use-proposal-preview";
import { presentProposal } from "./proposal-presenters";
import { classifyProposal, ranWithoutApproval, type ProposalPresentation } from "./proposal-state";

const TIME_FORMAT = { hour: "numeric", minute: "2-digit" } as const;

function useDecidedLine(
  state: ProposalPresentation,
  decidedAt: number | null | undefined,
  decidedBy: string,
  own: boolean,
): string {
  const t = useT();
  const me = useAuthStore((store) => store.user?.id ?? "");
  const time = decidedAt ? formatUnixInUserTimezone(decidedAt, TIME_FORMAT, "") : "";

  return recordStatus({ state, time, decidedByMe: me !== "" && decidedBy === me, own }, t);
}

/**
 * The one line a decision leaves in a conversation, opening onto what it
 * was. A record, never a question: what the agent proposed, where it stands
 * and who decided it, with the preview and the outcome behind the line. The
 * only place a waiting decision is answered is the approval box at the foot
 * of the conversation; its line says so.
 */
function DecisionRecordFrame({
  state,
  title,
  status,
  defaultOpen,
  children,
}: {
  state: ProposalPresentation;
  title: string;
  status: string;
  defaultOpen: boolean;
  children: (open: boolean) => ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const waiting = state === "awaiting";
  const Mark: LucideIcon | null = waiting ? HourglassIcon : null;

  return (
    <Collapsible
      open={open}
      onOpenChange={setOpen}
      data-slot="decision-record"
      data-state={state}
      className="min-w-0"
    >
      <CollapsibleTrigger className="hover:bg-surface-hover ui-focus-ring flex w-full min-w-0 items-center gap-2 rounded-md px-1.5 py-1 text-left text-xs transition-colors">
        {Mark ? (
          <Mark aria-hidden className="text-warning size-3.5 shrink-0" />
        ) : (
          <OutcomeIcon state={state} />
        )}
        <span className="min-w-0 flex-1 truncate">
          <span className="text-foreground">{title}</span>
          <span className="text-foreground-muted"> · {status}</span>
        </span>
        <ChevronRightIcon
          aria-hidden
          className={cn(
            "text-foreground-subtle size-3.5 shrink-0 transition-transform duration-200",
            open && "rotate-90",
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="ease-settle h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 data-ending-style:h-0 data-starting-style:h-0">
        <div className="border-border-subtle text-foreground-muted ml-3 flex min-w-0 flex-col gap-2 border-l pt-1.5 pb-1 pl-3 text-xs leading-relaxed">
          {children(open)}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/**
 * A proposal as the conversation keeps it: "Proposed: post invoice INV-104 ·
 * Approved by you 8:52 PM", opening onto what it would do or did, read-only.
 */
export function ProposalRecord({
  proposal,
  defaultOpen = false,
  showPreview = true,
}: {
  proposal: AssistantProposal;
  defaultOpen?: boolean;
  /** False where the surface around it already draws the preview in full. */
  showPreview?: boolean;
}) {
  const t = useT();
  const state = classifyProposal(proposal);
  const view = presentProposal(proposal);
  const status = useDecidedLine(
    state,
    proposal.decidedAt,
    proposal.decidedByUserId,
    ranWithoutApproval(proposal),
  );

  return (
    <DecisionRecordFrame
      state={state}
      title={t("Proposed: {0}", view.summary)}
      status={status}
      defaultOpen={defaultOpen}
    >
      {(open) => (
        <>
          <ProposedBy agentId={proposal.agentId} agentName={proposal.agentName} />
          <span className="block">
            <OutcomeLine proposal={proposal} state={state} />
          </span>
          <DecisionNoteLine note={proposal.decisionNote} />
          {showPreview && (
            <RecordedProposalPreview
              proposal={proposal}
              enabled={open}
              highlights={view.highlights}
            />
          )}
          {proposal.rationale !== "" && <p className="whitespace-pre-wrap">{proposal.rationale}</p>}
        </>
      )}
    </DecisionRecordFrame>
  );
}

/**
 * What the write would do, or did: read live while it waits, and as the
 * decision recorded it once decided. Read only while the record is open.
 */
function RecordedProposalPreview({
  proposal,
  enabled,
  highlights,
}: {
  proposal: AssistantProposal;
  enabled: boolean;
  highlights: { label: string; value: string }[];
}) {
  const query = useProposalPreview({ scope: "mine", id: proposal.id, enabled });
  if (!enabled) {
    return null;
  }

  return (
    <PreviewLoadState
      query={query}
      changed={false}
      density="compact"
      fallback={<Highlights highlights={highlights} />}
    >
      {(preview) => <ProposalPreview preview={preview} density="compact" />}
    </PreviewLoadState>
  );
}

/**
 * A plan as the conversation keeps it: its title and where it stands on one
 * line, opening onto its steps, each with what it would change while the
 * plan waits and what became of it once decided.
 */
export function PlanRecord({
  plan,
  steps,
  defaultOpen = false,
}: {
  plan: AssistantPlan;
  steps: AssistantProposal[];
  defaultOpen?: boolean;
}) {
  const t = useT();
  const state = classifyPlan(plan);
  const status = useDecidedLine(state, plan.decidedAt, plan.decidedByUserId, false);
  const note = steps.find((step) => step.decisionNote.trim() !== "")?.decisionNote ?? "";

  return (
    <DecisionRecordFrame
      state={state}
      title={t("Proposed: {0}", plan.title)}
      status={status}
      defaultOpen={defaultOpen}
    >
      {(open) => (
        <>
          <ProposedBy agentId={plan.agentId} agentName={plan.agentName} />
          <span className="block">
            {t("{0, plural, one {# change, in order} other {# changes, in order}}", plan.stepCount)}
          </span>
          <PlanOutcomeLine plan={plan} state={state} steps={steps} />
          <DecisionNoteLine note={note} />
          {plan.summary !== "" && <p className="whitespace-pre-wrap">{plan.summary}</p>}
          {steps.length > 0 && <PlanSteps plan={plan} steps={steps} state={state} enabled={open} />}
        </>
      )}
    </DecisionRecordFrame>
  );
}

function PlanSteps({
  plan,
  steps,
  state,
  enabled,
}: {
  plan: AssistantPlan;
  steps: AssistantProposal[];
  state: ProposalPresentation;
  enabled: boolean;
}) {
  const undecided = state === "awaiting" || state === "held";
  const query = usePlanPreview({ scope: "mine", id: plan.id, enabled: enabled && undecided });
  const previews = useMemo(() => previewsByStep(query.data), [query.data]);

  if (!undecided) {
    return <StepList steps={steps} settled />;
  }

  return (
    <div className="flex flex-col gap-2">
      <StepList steps={steps} settled={false} previews={previews} />
      {enabled && (
        <PreviewLoadState query={query} changed={false} density="compact">
          {(preview) =>
            preview.stale && !preview.steps.some((step) => step.preview.stale) ? (
              <StaleNotice missing={false} />
            ) : null
          }
        </PreviewLoadState>
      )}
    </div>
  );
}

/**
 * A decision the agent asked the person to make, as the Desk's pane keeps
 * it: the records of what was asked, read from the conversation's proposals,
 * and, while any of it waits, the way to the approval box where it is
 * decided. A step of a plan, or a plan asked for by id, is the plan's record.
 */
export function RequestedDecisionRecords({
  request,
  threadId,
}: {
  request: DecisionFocus;
  threadId: string;
}) {
  const t = useT();
  const focusDecision = useAssistantStore((state) => state.focusDecision);
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
    return <Skeleton className="h-16 w-full" aria-label={t("Loading the proposal")} />;
  }

  const waiting =
    plan !== null
      ? classifyPlan(plan) === "awaiting"
      : requested.some((proposal) => classifyProposal(proposal) === "awaiting");
  const decideHere = waiting ? (
    <Button
      size="sm"
      variant="outline"
      className="self-start"
      onClick={() => {
        const focus =
          plan !== null
            ? { proposalIds: [], planId: plan.id }
            : { proposalIds: requested.map((proposal) => proposal.id), planId: "" };
        focusDecision(threadId, focus, focusKeys(focus));
      }}
    >
      {t("Decide in the approval box")}
    </Button>
  ) : null;

  if (plan !== null) {
    return (
      <div className="flex flex-col gap-2">
        <PlanRecord plan={plan} steps={steps} defaultOpen />
        {decideHere}
      </div>
    );
  }

  if (requested.length === 0 || request.planId !== "") {
    return (
      <p className="text-muted-foreground text-xs">
        {t("This proposal is no longer part of the conversation.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col">
        {requested.map((proposal) => (
          <ProposalRecord
            key={proposal.id}
            proposal={proposal}
            defaultOpen={requested.length === 1}
          />
        ))}
      </div>
      {decideHere}
    </div>
  );
}
