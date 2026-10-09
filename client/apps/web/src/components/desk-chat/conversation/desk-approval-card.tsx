import { decideBatch } from "@/components/assistant/approval-actions";
import type { ApprovalEntry } from "@/components/assistant/approval-queue";
import { canApprove, gateDigest } from "@/components/assistant/proposal-preview/preview-gate";
import {
  useApprovalGate,
  usePlanPreview,
  useProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { decideMyPlan, decideMyProposal } from "@/lib/graphql/agent-decisions";
import {
  invalidateProposalViews,
  markPlanDecided,
  markPlanStatus,
  markProposalDecided,
  markProposalsStatus,
} from "@/lib/proposal-cache";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, type MutableRefObject } from "react";
import { toast } from "sonner";
import { DeskIcon } from "../desk-icons";
import { approvalFacts, recordCount, refusalReasons, type ApprovalFacts } from "./approval-facts";
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
  /** Kept pointing at this card's approve while it can approve, for ⌘↵. */
  approveRef: MutableRefObject<(() => void) | null>;
  /** Asks the agent something in the conversation, such as to redraft a stale change. */
  onAsk?: (text: string) => void;
};

function hold(undo: CardProps["undo"], held: Held) {
  undo.start({ ...held, commitsAt: null, startedAt: Date.now() });
}

/**
 * The change waiting on the person, as one compact row above the composer:
 * what it does and to how many records, the field it sets from → to, whether
 * it can be undone, and Review, Not now and Approve. ⌘↵ approves.
 */
export function DeskApprovalCard(props: CardProps) {
  switch (props.entry.kind) {
    case "proposal":
      return <ProposalCard {...props} entry={props.entry} />;
    case "batch":
      return <BatchCard {...props} entry={props.entry} />;
    default:
      return <PlanCard {...props} entry={props.entry} />;
  }
}

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
  const { proposal } = entry;
  const view = presentProposal(proposal);
  const previewQuery = useProposalPreview({ scope: "mine", id: proposal.id });
  const approval = useApprovalGate(previewQuery);
  const facts = approvalFacts(previewQuery.data, t);

  // Wording the person changed on the draft and sent for approval is kept on
  // the proposal, and goes with the approval as a modification of what the
  // agent proposed. Deciding clears it on the server.
  const edits =
    proposal.pendingModifications && Object.keys(proposal.pendingModifications).length > 0
      ? proposal.pendingModifications
      : null;
  const mutation = useMutation({
    mutationFn: () =>
      decideMyProposal(
        proposal.id,
        edits
          ? { decision: "Modified", reasonCode: "modified_from_desk", modifications: edits }
          : { decision: "Accepted", reasonCode: "", previewDigest: gateDigest(approval.gate) },
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
    onSuccess: (decision) => {
      const commitsAt = decision.commitsAt ?? null;
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
  const approvable = canApprove(approval.gate) && !mutation.isPending;
  const approve = () => {
    approval.acknowledge();
    mutation.mutate();
  };
  useApproveShortcut(approveRef, approve, approvable);

  return (
    <CardRow
      title={view.title}
      facts={facts}
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
  const { plan, steps } = entry;
  const previewQuery = usePlanPreview({ scope: "mine", id: plan.id });
  const approval = useApprovalGate(previewQuery);
  const title = plan.title || t("Run the plan");
  const facts: ApprovalFacts = {
    count: plan.stepCount,
    resource: "",
    field: null,
    refused: [],
    wouldFail: null,
    changedSince: 0,
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
    onSuccess: (decided) => {
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
  const approvable = canApprove(approval.gate) && !mutation.isPending;
  const approve = () => {
    approval.acknowledge();
    mutation.mutate();
  };
  useApproveShortcut(approveRef, approve, approvable);

  return (
    <CardRow
      title={title}
      facts={facts}
      countLabel={t("{0, plural, one {# step} other {# steps}}", plan.stepCount)}
      reversible={reversible}
      changed={approval.changed}
      approvable={approvable}
      onApprove={approve}
      onReview={() => onReview(entry)}
      onDefer={onDefer}
    />
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
  const facts: ApprovalFacts = { ...sample, count: proposals.length };
  const reversible = proposals.every((proposal) => presentProposal(proposal).reversible);

  const mutation = useMutation({
    mutationFn: () =>
      decideBatch(queryClient, proposals, { approving: true, shownDigests: new Map() }),
    onMutate: () =>
      hold(undo, {
        key: entry.key,
        title: view.title,
        what: approvedWhat(view.title, facts, t),
        target: { proposalId: first.id },
        proposalIds: proposals.map((proposal) => proposal.id),
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
  const approvable = !mutation.isPending;
  const approve = () => mutation.mutate();
  useApproveShortcut(approveRef, approve, approvable);

  return (
    <CardRow
      title={view.title}
      facts={facts}
      reversible={reversible}
      changed={false}
      approvable={approvable}
      onApprove={approve}
      onReview={() => onReview(entry)}
      onDefer={onDefer}
    />
  );
}

/** What an approval will do when its undo window closes, e.g. "Assign biller on 11 items". */
function approvedWhat(title: string, facts: ApprovalFacts, t: TranslateFn): string {
  // A presenter's title is English source text from the catalog; translate it here, where
  // it is put into the sentence.
  const shown = t(title);
  return facts.count > 0
    ? t("{0} on {1}", shown, recordCount(facts.resource, facts.count, t))
    : shown;
}

const cardButton = "rounded-lg in-[.dk-dense]:order-2";
const leadButton = "in-[.dk-dense]:ml-auto";
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
  stale = false,
  approvable,
  onApprove,
  onReview,
  onDefer,
  onRedraft,
}: {
  title: string;
  facts: ApprovalFacts;
  countLabel?: string;
  reversible: boolean;
  /** The change looks different from what was shown; read it again before approving. */
  changed: boolean;
  /** Records changed since the agent drafted it; the server will not take an approval. */
  stale?: boolean;
  approvable: boolean;
  onApprove: () => void;
  onReview: () => void;
  onDefer: () => void;
  onRedraft?: () => void;
}) {
  const t = useT();
  const scope =
    countLabel ?? (facts.count > 0 ? recordCount(facts.resource, facts.count, t) : null);

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
            {title} {scope && <span className="dk-dcx-s">{t("on {0}", scope)}</span>}
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
        <Button
          variant="quiet"
          size="sm"
          className={cn(plainButton, leadButton)}
          onClick={onDefer}
        >
          {t("Not now")}
        </Button>
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
        <Button
          variant="quiet"
          size="sm"
          className={cn(plainButton, leadButton)}
          onClick={onReview}
        >
          {refusedCount > 0 ? t("Review {0}", refusedCount) : t("Review")}
        </Button>
        {all ? (
          <Button variant="quiet" size="sm" className={plainButton} onClick={onDefer}>
            {t("Not now")}
          </Button>
        ) : (
          <Button size="sm" className={approveButton} disabled={!approvable} onClick={onApprove}>
            {t("Approve {0}, skip {1}", facts.count - refusedCount, refusedCount)}
          </Button>
        )}
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
          {title} {scope && <span className="dk-dcx-s">{t("on {0}", scope)}</span>}
        </b>
        <span className="dk-dcx-d">
          {changed ? (
            t("This changed since it was drafted. Read it again before approving.")
          ) : facts.field ? (
            <>
              {facts.field.label} <span className="sr-only">{t("from")}</span>
              <s>{facts.field.before}</s> <span aria-hidden>→</span>
              <span className="sr-only">{t("to")}</span> <em>{facts.field.after}</em>
            </>
          ) : null}
          <span className="dk-dcx-m">· {reversible ? t("reversible") : t("can't be undone")}</span>
        </span>
      </span>
      <Button
        variant="quiet"
        size="sm"
        className={cn(warnButton, leadButton)}
        onClick={onReview}
      >
        {t("Review")}
      </Button>
      <Button variant="quiet" size="sm" className={warnButton} onClick={onDefer}>
        {t("Not now")}
      </Button>
      <Button
        size="sm"
        className={cn("dk-apv-b", approveButton)}
        disabled={!approvable}
        aria-keyshortcuts="Meta+Enter"
        onClick={onApprove}
      >
        {t("Approve")}
        <span className="dk-kbd" aria-hidden>
          ⌘↵
        </span>
      </Button>
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
