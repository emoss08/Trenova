import { decideBatch, useAfterDecision } from "@/components/assistant/approval-actions";
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
  markProposalDecided,
} from "@/lib/proposal-cache";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useEffect, type MutableRefObject } from "react";
import { toast } from "sonner";
import { DeskIcon } from "../desk-icons";
import { approvalFacts, recordCount, refusalReasons, type ApprovalFacts } from "./approval-facts";

/** What the card said about a change, kept for the moment after it is approved. */
export type ApprovedNote = { key: string; title: string; detail: string };

type CardProps = {
  threadId: string;
  entry: ApprovalEntry;
  /** Opens the change in the workspace, where it can be read in full, edited or rejected. */
  onReview: (entry: ApprovalEntry) => void;
  /** Puts the change off; it waits in the workspace and in Decisions. */
  onDefer: () => void;
  onDecided: (entry: ApprovalEntry) => void;
  /** Told the moment the server takes the approval, with what to say while it runs. */
  onApproved: (note: ApprovedNote) => void;
  /** Kept pointing at this card's approve while it can approve, for ⌘↵. */
  approveRef: MutableRefObject<(() => void) | null>;
  /** Asks the agent something in the conversation, such as to redraft a stale change. */
  onAsk?: (text: string) => void;
};

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
  onApproved,
  approveRef,
  onAsk,
}: CardProps & { entry: Extract<ApprovalEntry, { kind: "proposal" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { proposal } = entry;
  const view = presentProposal(proposal);
  const previewQuery = useProposalPreview({ scope: "mine", id: proposal.id });
  const approval = useApprovalGate(previewQuery);
  const afterDecision = useAfterDecision(threadId, entry, onDecided);
  const facts = approvalFacts(previewQuery.data, t);

  const mutation = useMutation({
    mutationFn: () =>
      decideMyProposal(proposal.id, {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: gateDigest(approval.gate),
      }),
    onMutate: () => onApproved(approvedNote(entry.key, view.title, facts, t)),
    onSuccess: async () => {
      markProposalDecided(queryClient, proposal.id, "Accepted");
      await afterDecision(proposal.id);
    },
    onError: (error) => {
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
  onApproved,
  approveRef,
}: CardProps & { entry: Extract<ApprovalEntry, { kind: "plan" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { plan, steps } = entry;
  const previewQuery = usePlanPreview({ scope: "mine", id: plan.id });
  const approval = useApprovalGate(previewQuery);
  const afterDecision = useAfterDecision(threadId, entry, onDecided);
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
      onApproved({
        key: entry.key,
        title,
        detail: t("{0, plural, one {Running # step…} other {Running # steps…}}", plan.stepCount),
      }),
    onSuccess: async () => {
      markPlanDecided(queryClient, plan.id, "Accepted");
      await afterDecision(steps[0]?.id ?? plan.id);
    },
    onError: (error) => {
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
  onApproved,
  approveRef,
}: CardProps & { entry: Extract<ApprovalEntry, { kind: "batch" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { proposals } = entry;
  const first = proposals[0];
  const view = presentProposal(first);
  const previewQuery = useProposalPreview({ scope: "mine", id: first.id });
  const afterDecision = useAfterDecision(threadId, entry, onDecided);
  const sample = approvalFacts(previewQuery.data, t);
  const facts: ApprovalFacts = { ...sample, count: proposals.length };
  const reversible = proposals.every((proposal) => presentProposal(proposal).reversible);

  const mutation = useMutation({
    mutationFn: () =>
      decideBatch(queryClient, proposals, { approving: true, shownDigests: new Map() }),
    onMutate: () => onApproved(approvedNote(entry.key, view.title, facts, t)),
    onSuccess: async (outcome) => {
      if (outcome.errors.length > 0) {
        toast.warning(t("{0} of {1} went through", outcome.approved, outcome.total), {
          description: outcome.errors.join(" · "),
        });
      }
      await afterDecision(first.id);
    },
    onError: (error) => {
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

function approvedNote(
  key: string,
  title: string,
  facts: ApprovalFacts,
  t: TranslateFn,
): ApprovedNote {
  return {
    key,
    title,
    detail:
      facts.count > 0
        ? t("{0} on {1}…", title, recordCount(facts.resource, facts.count, t))
        : t("{0}…", title),
  };
}

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
        <button type="button" className="dk-bt dk-sm" onClick={onDefer}>
          {t("Not now")}
        </button>
        {onRedraft ? (
          <button type="button" className="dk-apv-b" onClick={onRedraft}>
            {facts.changedSince > 0 && going > 0
              ? t("Redraft with {0}", recordCount(facts.resource, going, t))
              : t("Redraft")}
          </button>
        ) : (
          <button type="button" className="dk-apv-b" onClick={onReview}>
            {t("Review")}
          </button>
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
      .map(([reason, n]) => `${n} ${reason.charAt(0).toLowerCase()}${reason.slice(1)}`);
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
        <button type="button" className="dk-bt dk-sm" onClick={onReview}>
          {refusedCount > 0 ? t("Review {0}", refusedCount) : t("Review")}
        </button>
        {all ? (
          <button type="button" className="dk-bt dk-sm" onClick={onDefer}>
            {t("Not now")}
          </button>
        ) : (
          <button
            type="button"
            className="dk-apv-b dk-ec-fix"
            disabled={!approvable}
            onClick={onApprove}
          >
            {t("Approve {0}, skip {1}", facts.count - refusedCount, refusedCount)}
          </button>
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
              {facts.field.label} <s>{facts.field.before}</s> → <em>{facts.field.after}</em>
            </>
          ) : null}
          <span className="dk-dcx-m">· {reversible ? t("reversible") : t("can't be undone")}</span>
        </span>
      </span>
      <button type="button" className="dk-bt dk-sm" onClick={onReview}>
        {t("Review")}
      </button>
      <button type="button" className="dk-bt dk-sm" onClick={onDefer}>
        {t("Not now")}
      </button>
      <button type="button" className="dk-apv-b" disabled={!approvable} onClick={onApprove}>
        {t("Approve")}
        <span className="dk-kbd">⌘↵</span>
      </button>
    </div>
  );
}

/** The card once approved: a green row that says what is now running, then leaves. */
export function DeskApprovedCard({ note }: { note: ApprovedNote }) {
  const t = useT();

  return (
    <div className="dk-dcx dk-ok" key={note.key} role="status">
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
        <span>{note.detail}</span>
      </span>
      <span className="dk-shine" />
    </div>
  );
}
