import { CONVERSATION_REJECT_REASON, decideBatch } from "@/components/assistant/approval-actions";
import { useDecisionFollowUp } from "@/components/assistant/decision-follow-up";
import {
  holdActionText,
  holdReasonText,
  holdSwitch,
  pendingEdits,
  type ApprovalEntry,
} from "@/components/assistant/approval-queue";
import {
  canApprove,
  gateDigest,
  reviewRequiredFor,
  unreviewedRequired,
} from "@/components/assistant/proposal-preview/preview-gate";
import {
  ReviewBlocker,
  ReviewedChange,
  reviewTally,
  useReviewReports,
} from "@/components/assistant/proposal-preview/reviewed-change";
import {
  useApprovalGate,
  usePlanPreview,
  useProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { decideMyPlan, decideMyProposal } from "@/lib/graphql/agent-decisions";
import { announceRefusal, approvalFailure } from "@/lib/approval-outcome";
import {
  invalidateProposalViews,
  markPlanDecided,
  markPlanFailed,
  markPlanStatus,
  markProposalDecided,
  markProposalFailed,
  markProposalsStatus,
} from "@/lib/proposal-cache";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ReasonDialog } from "@/components/reason-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState, type MutableRefObject } from "react";
import { Link } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { DeskIcon, type DeskIconName } from "../desk-icons";
import {
  approvalFacts,
  recordCount,
  refusalReasons,
  scopePhrase,
  type ApprovalFacts,
} from "./approval-facts";
import type { UndoController } from "./desk-undo-bar";
import type { UndoWindow } from "./undo-window";

/** What the card hands the undo window when it is approved. */
type Held = Omit<UndoWindow, "commitsAt" | "startedAt">;

type CardProps = {
  threadId: string;
  entry: ApprovalEntry;
  /** Opens the change in the workspace, where it can be read in full, edited or rejected. */
  onReview: (entry: ApprovalEntry) => void;
  /** Puts the change off; it waits in the workspace and in Decisions. */
  onDefer: () => void;
  onDecided: (entry: ApprovalEntry) => void;
  /**
   * The undo window an approval opens: started on the click, scheduled when
   * the server says when it commits, cleared if the server refuses it.
   */
  undo: Pick<UndoController, "start" | "scheduled" | "clear">;
  /** Kept pointing at this card's approve while it can approve, for Cmd/Ctrl+Enter. */
  approveRef: MutableRefObject<(() => void) | null>;
  /** Asks the agent something in the conversation, such as to redraft a stale change. */
  onAsk?: (text: string) => void;
};

function hold(undo: CardProps["undo"], held: Held) {
  undo.start({ ...held, commitsAt: null, startedAt: Date.now() });
}

/**
 * The change waiting on the person, as one compact row above the composer:
 * what it does and to which record or how many, the field it sets from → to,
 * Permanent when it can't be undone, and Review, Decline, Decide later and
 * Approve. Cmd/Ctrl+Enter approves from an empty message box.
 */
export function DeskApprovalCard(props: CardProps) {
  if (props.entry.hold !== null) {
    return <HeldCard entry={props.entry} onReview={props.onReview} onDefer={props.onDefer} />;
  }

  switch (props.entry.kind) {
    case "proposal":
      return <ProposalCard {...props} entry={props.entry} />;
    case "batch":
      return <BatchCard {...props} entry={props.entry} />;
    default:
      return <PlanCard {...props} entry={props.entry} />;
  }
}

/**
 * Declining from the card: the reason dialog Decisions uses, required, and the
 * decision sent with it as the note the agent's follow-up turn reads.
 */
function useDecline(count: number, decline: (note: string) => Promise<void>) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const mutation = useMutation({
    mutationFn: decline,
    onSuccess: () => setOpen(false),
    onError: (error) => handleMutationError({ error, resourceName: "Proposal" }),
  });

  const dialog = (
    <ReasonDialog
      open={open}
      title={count > 1 ? t("Decline {0} changes?", count) : t("Decline this change?")}
      description={t("Say why. The agent sees it, and so does whoever looks after the agent.")}
      placeholder={t("What's wrong with it, or what to do instead")}
      confirmLabel={count > 1 ? t("Decline {0}", count) : t("Decline")}
      label={t("Reason")}
      maxLength={DECLINE_NOTE_LENGTH}
      destructive
      pending={mutation.isPending}
      onConfirm={(reason) => mutation.mutate(reason)}
      onOpenChange={setOpen}
    />
  );

  return { open: () => setOpen(true), dialog, pending: mutation.isPending };
}

/** The longest note the server keeps with a decision. */
const DECLINE_NOTE_LENGTH = 2000;

function useApproveShortcut(
  approveRef: MutableRefObject<(() => void) | null>,
  approve: () => void,
  ready: boolean,
) {
  useEffect(() => {
    approveRef.current = ready ? approve : null;
  });
  useEffect(() => () => void (approveRef.current = null), [approveRef]);
}

function ProposalCard({
  threadId,
  entry,
  onReview,
  onDefer,
  onDecided,
  undo,
  approveRef,
  onAsk,
}: CardProps & { entry: Extract<ApprovalEntry, { kind: "proposal" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const followUp = useDecisionFollowUp();
  const { proposal } = entry;
  const view = presentProposal(proposal);
  // Wording the person changed on the draft and sent for approval is kept on
  // the proposal, and goes with the approval as a modification of what the
  // agent proposed. Deciding clears it on the server. The preview is of the
  // change with those edits, so its digest is what the approval carries.
  const edits = pendingEdits(proposal);
  const previewQuery = useProposalPreview({
    scope: "mine",
    id: proposal.id,
    modifications: edits,
  });
  const approval = useApprovalGate(previewQuery);
  const facts = approvalFacts(previewQuery.data, t);
  const previewDigest = gateDigest(approval.gate);

  const mutation = useMutation({
    mutationFn: () =>
      decideMyProposal(
        proposal.id,
        edits
          ? {
              decision: "Modified",
              reasonCode: "modified_from_desk",
              modifications: edits,
              previewDigest,
            }
          : { decision: "Accepted", reasonCode: "", previewDigest },
      ),
    onMutate: () =>
      hold(undo, {
        key: entry.key,
        title: view.title,
        what: approvedWhat(view.title, facts, t),
        target: { proposalId: proposal.id },
        proposalIds: [proposal.id],
        planId: null,
      }),
    onSuccess: (result) => {
      // The approval is recorded, but its write did not go through: refused
      // because the record moved on, or failed as it ran. Nothing is held for
      // undoing and nothing reads as accepted; the card shows the failure and
      // the conversation picks up the turn the server started to say why.
      const failure = approvalFailure(result);
      if (failure !== null) {
        undo.clear(entry.key);
        markProposalFailed(queryClient, proposal.id, failure);
        if (result.refused) {
          announceRefusal(t, failure);
        }
        followUp?.(proposal.id);
        void invalidateProposalViews(queryClient, threadId);
        onDecided(entry);
        return;
      }
      const commitsAt = result.decision?.commitsAt ?? null;
      if (commitsAt !== null) {
        markProposalsStatus(queryClient, [proposal.id], "Approving");
      } else {
        markProposalDecided(queryClient, proposal.id, "Accepted");
      }
      undo.scheduled(entry.key, commitsAt);
      onDecided(entry);
    },
    onError: (error) => {
      undo.clear(entry.key);
      if (!approval.handleDecisionError(error)) {
        handleMutationError({ error, resourceName: "Proposal" });
      }
      void invalidateProposalViews(queryClient, threadId);
    },
  });
  const reviewRequired = reviewRequiredFor(
    previewQuery.data,
    proposal.reviewRequired,
    edits !== null,
  );
  const decline = useDecline(1, async (note) => {
    await decideMyProposal(proposal.id, {
      decision: "Rejected",
      reasonCode: CONVERSATION_REJECT_REASON,
      note,
    });
    markProposalDecided(queryClient, proposal.id, "Rejected");
    followUp?.(proposal.id);
    void invalidateProposalViews(queryClient, threadId);
    onDecided(entry);
  });
  const approvable =
    canApprove(approval.gate, reviewRequired) && !mutation.isPending && !decline.pending;
  const approve = () => {
    approval.acknowledge();
    mutation.mutate();
  };
  useApproveShortcut(approveRef, approve, approvable);

  return (
    <>
      <CardRow
        title={view.title}
        facts={facts}
        onDecline={decline.open}
        unreviewable={approval.gate.state === "unavailable" && reviewRequired}
        reversible={view.reversible}
        changed={approval.changed}
        stale={approval.gate.state === "stale"}
        approvable={approvable}
        onApprove={approve}
        onReview={() => onReview(entry)}
        onDefer={onDefer}
        onRedraft={
          onAsk
            ? () => onAsk(t("Draft that change again with the records as they are now."))
            : undefined
        }
      />
      {decline.dialog}
    </>
  );
}

function PlanCard({
  threadId,
  entry,
  onReview,
  onDefer,
  onDecided,
  undo,
  approveRef,
}: CardProps & { entry: Extract<ApprovalEntry, { kind: "plan" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const followUp = useDecisionFollowUp();
  const { plan, steps } = entry;
  const previewQuery = usePlanPreview({ scope: "mine", id: plan.id });
  const approval = useApprovalGate(previewQuery);
  const title = plan.title || t("Run the plan");
  const facts: ApprovalFacts = {
    count: plan.stepCount,
    resource: "",
    record: "",
    field: null,
    refused: [],
    wouldFail: null,
    changedSince: 0,
    creates: false,
  };
  const reversible = steps.every((step) => presentProposal(step).reversible);

  const mutation = useMutation({
    mutationFn: () =>
      decideMyPlan(plan.id, {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: gateDigest(approval.gate),
      }),
    onMutate: () =>
      hold(undo, {
        key: entry.key,
        title,
        what: t("{0, plural, one {Running # step} other {Running # steps}}", plan.stepCount),
        target: { planId: plan.id },
        proposalIds: steps.map((step) => step.id),
        planId: plan.id,
      }),
    onSuccess: ({ plan: decided, refused }) => {
      // A plan that failed as it was decided, refused because a record a
      // step changes moved on or stopped at a step that failed, is not held
      // for undoing and does not read as approved.
      if (decided.status === "Failed") {
        undo.clear(entry.key);
        markPlanFailed(queryClient, decided);
        if (refused) {
          announceRefusal(t, decided.failureError);
        }
        followUp?.(plan.id);
        void invalidateProposalViews(queryClient, threadId);
        onDecided(entry);
        return;
      }
      const commitsAt = decided.commitsAt ?? null;
      if (commitsAt !== null) {
        markPlanStatus(queryClient, plan.id, "Approving");
      } else {
        markPlanDecided(queryClient, plan.id, "Accepted");
      }
      undo.scheduled(entry.key, commitsAt);
      onDecided(entry);
    },
    onError: (error) => {
      undo.clear(entry.key);
      if (!approval.handleDecisionError(error)) {
        handleMutationError({ error, resourceName: "Plan" });
      }
      void invalidateProposalViews(queryClient, threadId);
    },
  });
  // Steps not listed yet say nothing about the plan, so it is held as one that
  // must be reviewed until its preview says otherwise.
  const reviewRequired = reviewRequiredFor(
    previewQuery.data,
    steps.length === 0 || steps.some((step) => step.reviewRequired),
    false,
  );
  const decline = useDecline(1, async (note) => {
    await decideMyPlan(plan.id, {
      decision: "Rejected",
      reasonCode: CONVERSATION_REJECT_REASON,
      note,
    });
    markPlanDecided(queryClient, plan.id, "Rejected");
    followUp?.(plan.id);
    void invalidateProposalViews(queryClient, threadId);
    onDecided(entry);
  });
  const approvable =
    canApprove(approval.gate, reviewRequired) && !mutation.isPending && !decline.pending;
  const approve = () => {
    approval.acknowledge();
    mutation.mutate();
  };
  useApproveShortcut(approveRef, approve, approvable);

  return (
    <>
      <CardRow
        title={title}
        facts={facts}
        onDecline={decline.open}
        unreviewable={approval.gate.state === "unavailable" && reviewRequired}
        countLabel={t("{0, plural, one {# step} other {# steps}}", plan.stepCount)}
        reversible={reversible}
        changed={approval.changed}
        approvable={approvable}
        onApprove={approve}
        onReview={() => onReview(entry)}
        onDefer={onDefer}
      />
      {decline.dialog}
    </>
  );
}

function BatchCard({
  threadId,
  entry,
  onReview,
  onDefer,
  onDecided,
  undo,
  approveRef,
}: CardProps & { entry: Extract<ApprovalEntry, { kind: "batch" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { proposals } = entry;
  const first = proposals[0];
  const view = presentProposal(first);
  const previewQuery = useProposalPreview({ scope: "mine", id: first.id });
  const sample = approvalFacts(previewQuery.data, t);
  // Only the first change's preview is on this row. A change the server takes
  // only once its preview has been shown is read here first: Review opens
  // each of their previews under the row, and the batch carries their digests.
  // The rest may be approved from here, unreviewed. A change whose record moved
  // since it was drafted can no longer be approved, so it is left out of the
  // approval and stays waiting for the person to set aside or redraft.
  const [reviewing, setReviewing] = useState(false);
  const { reports, shown, onReport } = useReviewReports();
  const gated = proposals.filter((proposal) => proposal.reviewRequired);
  const approving = proposals.filter((proposal) => reports.get(proposal.id)?.state !== "stale");
  const leftOut = proposals.length - approving.length;
  const unreviewed = unreviewedRequired(approving, shown);
  const needsReview = unreviewed.length;
  const tally = reviewTally(
    unreviewedRequired(proposals, shown).map((proposal) => proposal.id),
    reports,
  );
  const facts: ApprovalFacts = { ...sample, count: approving.length };
  const reversible = approving.every((proposal) => presentProposal(proposal).reversible);

  const mutation = useMutation({
    mutationFn: () => decideBatch(queryClient, approving, { approving: true, shownDigests: shown }),
    onMutate: () =>
      hold(undo, {
        key: entry.key,
        title: view.title,
        what: approvedWhat(view.title, facts, t),
        target: { proposalId: first.id },
        proposalIds: approving.map((proposal) => proposal.id),
        planId: null,
      }),
    onSuccess: (outcome) => {
      if (outcome.errors.length > 0) {
        toast.warning(t("{0} of {1} went through", outcome.approved, outcome.total), {
          description: outcome.errors.join(" · "),
        });
      }
      if (outcome.approved === 0) {
        undo.clear(entry.key);
        void invalidateProposalViews(queryClient, threadId);
        return;
      }
      undo.scheduled(entry.key, outcome.commitsAt);
      onDecided(entry);
    },
    onError: (error) => {
      undo.clear(entry.key);
      handleMutationError({ error, resourceName: "Proposals" });
      void invalidateProposalViews(queryClient, threadId);
    },
  });
  const decline = useDecline(proposals.length, async (note) => {
    const outcome = await decideBatch(queryClient, proposals, {
      approving: false,
      shownDigests: shown,
      note,
    });
    if (outcome.errors.length > 0) {
      toast.warning(t("{0} of {1} were declined", outcome.approved, outcome.total), {
        description: outcome.errors.join(" · "),
      });
    }
    void invalidateProposalViews(queryClient, threadId);
    if (outcome.approved > 0) {
      onDecided(entry);
    }
  });
  const approvable =
    needsReview === 0 && approving.length > 0 && !mutation.isPending && !decline.pending;
  const approve = () => mutation.mutate();
  useApproveShortcut(approveRef, approve, approvable);

  return (
    <>
      <CardRow
        title={view.title}
        facts={facts}
        reviewFirst={needsReview}
        reviewOpen={reviewing}
        leftOut={leftOut}
        reversible={reversible}
        changed={false}
        approvable={approvable}
        onApprove={approve}
        onReview={needsReview > 0 && !reviewing ? () => setReviewing(true) : () => onReview(entry)}
        onDefer={onDefer}
        onDecline={decline.open}
      />
      {decline.dialog}
      {reviewing && gated.length > 0 && (
        <div
          className="flex min-w-0 flex-col gap-3 px-3 pt-1 pb-2"
          role="group"
          aria-label={t("Changes to review before approving")}
        >
          {gated.map((proposal) => (
            <ReviewedChange key={proposal.id} proposal={proposal} onReport={onReport} />
          ))}
          <ReviewBlocker tally={tally} />
        </div>
      )}
    </>
  );
}

/**
 * A change that waits on the person while a switch keeps anyone from deciding
 * it: what it is, which switch holds it, and where that switch is turned off
 * for someone who may turn it off. There is nothing to approve, so Cmd/Ctrl+Enter does
 * nothing; the card takes its buttons once the switch is off and the
 * proposals are read again.
 */
function HeldCard({ entry, onReview, onDefer }: Pick<CardProps, "entry" | "onReview" | "onDefer">) {
  const t = useT();
  const held = holdSwitch(entry);
  const { allowed: canResume } = usePermission(Resource.AgentControl, Operation.Update);
  const { allowed: canEditAgent } = usePermission(Resource.AgentDefinition, Operation.Update);
  const { title, scope } = heldSummary(entry, t);

  if (held === null) {
    return null;
  }
  const canTurnOff = held.scope === "organization" ? canResume : canEditAgent;
  const why = holdReasonText(held, t);

  return (
    <div className="dk-dcx dk-ec-held" role="group" aria-label={t("Held, not yet decidable")}>
      <span className="dk-dcx-i">
        <DeskIcon name="pause" size={15} stroke={2} />
      </span>
      <span className="dk-dcx-t">
        <b>
          {title} {scope && <span className="dk-dcx-s">{scope}</span>}
        </b>
        <span className="dk-dcx-d">
          {why}
          {!canTurnOff && (
            <span className="dk-dcx-m">· {t("An administrator can turn this off.")}</span>
          )}
        </span>
      </span>
      <div className="dk-dcx-acts">
        <MoreActions
          actions={[
            { label: t("Review the change"), icon: "eye", onSelect: () => onReview(entry) },
            { label: t("Decide later"), icon: "clock", onSelect: onDefer },
          ]}
        />
        {canTurnOff && (
          <Link
            to={held.to}
            data-slot="button"
            className={cn(buttonVariants({ size: "sm" }), approveButton)}
          >
            {holdActionText(held, t)}
          </Link>
        )}
      </div>
    </div>
  );
}

/** The held card's title and scope: the change, the batch or the plan, and how much of it. */
function heldSummary(entry: ApprovalEntry, t: TranslateFn): { title: string; scope: string } {
  switch (entry.kind) {
    case "proposal":
      return { title: t(presentProposal(entry.proposal).title), scope: "" };
    case "batch":
      return {
        title: t(presentProposal(entry.proposals[0]).title),
        scope: t("{0, plural, one {# change} other {# changes}}", entry.proposals.length),
      };
    default:
      return {
        title: entry.plan.title || t("Run the plan"),
        scope: t("{0, plural, one {# step} other {# steps}}", entry.plan.stepCount),
      };
  }
}

/** What an approval will do when its undo window closes, e.g. "Assign biller on 11 items". */
function approvedWhat(title: string, facts: ApprovalFacts, t: TranslateFn): string {
  // A presenter's title is English source text from the catalog; translate it here, where
  // it is put into the sentence.
  const shown = t(title);
  const scope = scopePhrase(facts, t);
  return scope ? `${shown} ${scope}` : shown;
}

/** Puts the change off: it keeps waiting in the workspace and in Decisions. */
function DeferButton({ onDefer }: { onDefer: () => void }) {
  const t = useT();
  return (
    <Button
      variant="quiet"
      size="icon-sm"
      className={warnButton}
      aria-label={t("Decide later")}
      title={t("Decide later")}
      onClick={onDefer}
    >
      <DeskIcon name="clock" size={15} stroke={2} />
    </Button>
  );
}

/**
 * Says the change can't be taken back, beside the actions so nothing cuts it off; the
 * tooltip says what that means for the person about to approve.
 */
function PermanentPill() {
  const t = useT();
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="dk-dcx-perm ui-focus-ring" tabIndex={0}>
            {t("Permanent")}
          </span>
        }
      />
      <TooltipContent className="max-w-64">
        {t(
          "Once approved, this change can't be undone. Undo and the undo window don't apply to it.",
        )}
      </TooltipContent>
    </Tooltip>
  );
}

type MoreAction = { label: string; icon: DeskIconName; onSelect: () => void };

/**
 * The actions a card does not need on its face. The face keeps the decision (Decline and
 * Approve, or Review first); looking closer and putting it off live here, so the actions
 * share the title's row instead of taking a row of their own.
 */
function MoreActions({ actions }: { actions: readonly MoreAction[] }) {
  const t = useT();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="quiet"
            size="icon-sm"
            className={warnButton}
            aria-label={t("More actions")}
            title={t("More actions")}
          />
        }
      >
        <DeskIcon name="more" size={15} stroke={2} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="w-44">
        {actions.map((action) => (
          <DropdownMenuItem
            key={action.label}
            title={action.label}
            startContent={<DeskIcon name={action.icon} size={14} />}
            onClick={action.onSelect}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const cardButton = "rounded-lg";
const warnButton = cn(cardButton, "text-dsk-warn-fg hover:bg-dsk-warn/18 hover:text-dsk-fg");
const plainButton = cn(cardButton, "text-dsk-fg2 hover:bg-dsk-warn/18 hover:text-dsk-fg2");
const approveButton = cn(
  cardButton,
  "gap-2 bg-dsk-ink pr-1.5 pl-3 text-dsk-ink-fg hover:bg-dsk-ink hover:opacity-90 active:bg-dsk-ink",
);

function CardRow({
  title,
  facts,
  countLabel,
  reversible,
  changed,
  unreviewable = false,
  reviewFirst = 0,
  reviewOpen = false,
  leftOut = 0,
  stale = false,
  approvable,
  onApprove,
  onReview,
  onDefer,
  onRedraft,
  onDecline,
}: {
  title: string;
  facts: ApprovalFacts;
  countLabel?: string;
  reversible: boolean;
  /** The change looks different from what was shown; read it again before approving. */
  changed: boolean;
  /** Records changed since the agent drafted it; the server will not take an approval. */
  stale?: boolean;
  /** Its preview could not be read, and the server approves it only once one is shown. */
  unreviewable?: boolean;
  /** How many changes of a batch must have their preview read in Review before approving. */
  reviewFirst?: number;
  /** Their previews are open under the row; Review opens the decision instead. */
  reviewOpen?: boolean;
  /** How many changes of a batch moved since they were drafted and are left out of it. */
  leftOut?: number;
  approvable: boolean;
  onApprove: () => void;
  onReview: () => void;
  onDefer: () => void;
  onRedraft?: () => void;
  /** Turns the change down with a reason; absent where declining is not offered. */
  onDecline?: () => void;
}) {
  const t = useT();
  const scope = countLabel ? t("on {0}", countLabel) : scopePhrase(facts, t);

  // Out of date: what it was drafted against has changed, so it is drafted
  // again rather than approved.
  if (stale) {
    const going = Math.max(0, facts.count - facts.changedSince);
    return (
      <div className="dk-dcx dk-ec-stale" role="group" aria-label={t("Out of date")}>
        <span className="dk-dcx-i">
          <DeskIcon name="undo" size={15} stroke={2} />
        </span>
        <span className="dk-dcx-t">
          <b>
            {title} {scope && <span className="dk-dcx-s">{scope}</span>}
          </b>
          <span className="dk-dcx-d">
            {facts.changedSince > 0
              ? t(
                  "{0, plural, one {# of these items changed after it was drafted} other {# of these items changed after it was drafted}}",
                  facts.changedSince,
                )
              : t("What it was drafted against has changed since")}
          </span>
        </span>
        <div className="dk-dcx-acts">
          <DeferButton onDefer={onDefer} />
          {onRedraft ? (
            <Button size="sm" className={approveButton} onClick={onRedraft}>
              {facts.changedSince > 0 && going > 0
                ? t("Redraft with {0}", recordCount(facts.resource, going, t))
                : t("Redraft")}
            </Button>
          ) : (
            <Button size="sm" className={approveButton} onClick={onReview}>
              {t("Review")}
            </Button>
          )}
        </div>
      </div>
    );
  }

  // Would be refused: all of it, or the records named in the preview. The
  // rest go through on approval; the refused ones are listed afterwards.
  const refusedCount = facts.refused.length;
  if (facts.wouldFail !== null || refusedCount > 0) {
    const all = facts.wouldFail !== null || refusedCount >= facts.count;
    const reasons = refusalReasons(facts.refused)
      .slice(0, 3)
      .map(([reason, n]) => t("{0} ({1})", reason, n));
    return (
      <div className="dk-dcx dk-ec-would" role="group" aria-label={t("Would be refused")}>
        <span className="dk-dcx-i">
          <DeskIcon name="alert" size={15} stroke={2} />
        </span>
        <span className="dk-dcx-t">
          <b>
            {title} <span className="dk-dcx-s">{t("would be refused as it stands")}</span>
          </b>
          <span className="dk-dcx-d">
            {refusedCount > 0
              ? [t("{0} of {1}", refusedCount, facts.count), ...reasons].join(" · ")
              : (facts.wouldFail ?? "")}
          </span>
        </span>
        <div className="dk-dcx-acts">
          <MoreActions
            actions={[
              {
                label: refusedCount > 0 ? t("Review {0}", refusedCount) : t("Review"),
                icon: "eye",
                onSelect: onReview,
              },
              { label: t("Decide later"), icon: "clock", onSelect: onDefer },
            ]}
          />
          {onDecline && (
            <Button variant="quiet" size="sm" className={plainButton} onClick={onDecline}>
              {t("Decline")}
            </Button>
          )}
          {!all && (
            <Button size="sm" className={approveButton} disabled={!approvable} onClick={onApprove}>
              {t("Approve {0}, skip {1}", facts.count - refusedCount, refusedCount)}
            </Button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="dk-dcx" role="group" aria-label={t("Waiting on your approval")}>
      <span className="dk-dcx-i">
        <DeskIcon name="info" size={15} stroke={2} />
      </span>
      <span className="dk-dcx-t">
        <b>
          {title} {scope && <span className="dk-dcx-s">{scope}</span>}
        </b>
        <span className="dk-dcx-d">
          {changed ? (
            t("This changed since it was drafted. Read it again before approving.")
          ) : unreviewable ? (
            t(
              "Its preview could not be read, and this change is approved only once you've seen it.",
            )
          ) : reviewFirst > 0 ? (
            t(
              "{0, plural, one {Review # change before approving: it is approved only once you've seen its preview} other {Review # changes before approving: each is approved only once you've seen its preview}}",
              reviewFirst,
            )
          ) : facts.field ? (
            <>
              {facts.field.label} <span className="sr-only">{t("from")}</span>
              <s>{facts.field.before}</s> <span aria-hidden>→</span>
              <span className="sr-only">{t("to")}</span> <em>{facts.field.after}</em>
            </>
          ) : null}
          {reversible && <span className="dk-dcx-m">· {t("reversible")}</span>}
          {leftOut > 0 && (
            <span className="dk-dcx-m">
              ·{" "}
              {t(
                "{0, plural, one {# left out: it changed since it was drafted} other {# left out: they changed since they were drafted}}",
                leftOut,
              )}
            </span>
          )}
        </span>
      </span>
      <div className="dk-dcx-acts">
        {!reversible && <PermanentPill />}
        {reviewFirst > 0 ? (
          <>
            <DeferButton onDefer={onDefer} />
            {onDecline && (
              <Button variant="quiet" size="sm" className={warnButton} onClick={onDecline}>
                {t("Decline")}
              </Button>
            )}
            <Button
              size="sm"
              className={approveButton}
              onClick={onReview}
              aria-expanded={reviewOpen}
            >
              {reviewOpen ? t("Open the decision") : t("Review {0}", reviewFirst)}
            </Button>
          </>
        ) : (
          <>
            <MoreActions
              actions={[
                { label: t("Review the change"), icon: "eye", onSelect: onReview },
                { label: t("Decide later"), icon: "clock", onSelect: onDefer },
              ]}
            />
            {onDecline && (
              <Button variant="quiet" size="sm" className={warnButton} onClick={onDecline}>
                {t("Decline")}
              </Button>
            )}
            <Button
              size="sm"
              className={cn("dk-apv-b", approveButton)}
              disabled={!approvable}
              aria-keyshortcuts="Meta+Enter Control+Enter"
              onClick={onApprove}
            >
              {t("Approve")}
              <span className="dk-kbd" aria-hidden>
                {formatShortcut("↵")}
              </span>
            </Button>
          </>
        )}
      </div>
    </div>
  );
}

/** The approval once it has gone through: a green row that says what is now running, then leaves. */
export function DeskApprovedCard({ held }: { held: UndoWindow }) {
  const t = useT();

  return (
    <div className="dk-dcx dk-ok" key={held.key} role="status">
      <span className="dk-okr">
        <svg
          width="11"
          height="11"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="3.2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden
        >
          <path d="M5 12.5l4.5 4.5L19 7" />
        </svg>
      </span>
      <span className="dk-dcx-t">
        <b>{t("Approved")}</b>
        <span>{t("{0}…", held.what)}</span>
      </span>
      <span className="dk-shine" />
    </div>
  );
}
