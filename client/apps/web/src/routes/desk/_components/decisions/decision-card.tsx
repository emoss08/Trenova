import { PlanPreview } from "@/components/assistant/proposal-preview/plan-preview";
import {
  canApprove,
  gateDigest,
  reviewRequiredFor,
} from "@/components/assistant/proposal-preview/preview-gate";
import { askAgentPlanMessage } from "@/components/assistant/proposal-preview/preview-warnings";
import {
  PreviewLoadState,
  type WouldFailActions,
} from "@/components/assistant/proposal-preview/proposal-preview";
import {
  useApprovalGate,
  usePlanPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { conversationPath } from "@/lib/conversation-path";
import type { ProposalPreview as ProposalPreviewData } from "@/lib/graphql/agent-preview";
import { agentRunPath } from "@/lib/record-paths";
import { formatTimeAgo } from "@/lib/time-utils";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useEffect, useMemo } from "react";
import { Link } from "react-router";
import {
  approvalFacts,
  OUTCOME_FIELD,
  recordName,
  recordCount,
  refusalReasons,
  REFUSED_PREFIX,
  valueText,
} from "@/components/desk-chat/conversation/approval-facts";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { asAssistantProposal } from "./decision-presenters";
import {
  usePlanSteps,
  type PendingDecisionNode,
  type PendingPlanNode,
} from "./use-pending-decisions";
import { useRichT } from "@trenova/shared/i18n/rich";
import { QueueCard } from "../queue/queue";
import type { QueueLeave } from "../queue/use-queue-cursor";

/** What the card tells the bar about the decision on screen. */
export type CardGate = {
  /** False while the preview is first read, and once the record has moved. */
  approvable: boolean;
  /** The digest of the preview on screen, sent with the decision. */
  digest: string | undefined;
  /** Records the write would refuse as they stand. */
  refused: number;
  /** Records the write touches. */
  count: number;
  /** Reads the preview again when a decision found it out of date; true when it did. */
  handleError: (error: unknown) => boolean;
};

export type CardProps = {
  node: PendingPlanNode;
  leaving: QueueLeave | null;
  onGate: (gate: CardGate) => void;
  /** Turns the plan down with a reason already written. */
  onDeclineWith: (reason: string) => void;
};

export function Who({ node }: { node: PendingDecisionNode }) {
  const t = useT();
  const rt = useRichT();
  const run = node.run;
  const definition = run?.definition ?? null;
  const conversationId = run?.subjectType === "AssistantThread" ? run.subjectId : null;

  return (
    <div className="dk-dc2-who">
      <DeskAgentTile agent={definition} size="sm" />
      <span>
        {rt("<b>{0}</b> wants to:", { b: (c) => <b>{c}</b> }, definition?.name ?? t("An agent"))}
      </span>
      <span className="dk-dc2-ago">{formatTimeAgo(node.createdAt * 1000)}</span>
      {conversationId ? (
        <Link className="dk-ec-link" to={conversationPath(conversationId)}>
          {t("See the conversation")}
        </Link>
      ) : run ? (
        <Link className="dk-ec-link" to={agentRunPath(run.id)}>
          {t("See the run")}
        </Link>
      ) : null}
    </div>
  );
}

export type DiffRow = { key: string; before: string; after: string; refused: string | null };

/**
 * The records a write touches, one line each with the value it sets, when
 * every record has the same field set; null when the full preview should
 * draw it instead.
 */
export function diffRows(preview: ProposalPreviewData, t: TranslateFn): DiffRow[] | null {
  const changes = preview.changes.filter((change) => !change.withheld);
  const path = changes[0]?.fields.find(
    (field) => !field.withheld && field.path !== OUTCOME_FIELD,
  )?.path;
  // A write that would fail as a whole, creates or removes records, sends a
  // message or moves money is drawn whole: its preview carries the ways
  // forward and the detail a single line would lose.
  if (
    !path ||
    preview.warnings.some((warning) => warning.code === "would_fail") ||
    changes.some((change) => change.message || change.money || change.operation !== "Update")
  ) {
    return null;
  }
  const rows: DiffRow[] = [];
  for (const change of changes) {
    const field = change.fields.find((candidate) => candidate.path === path);
    if (!field) {
      return null;
    }
    const outcome = change.fields.find((candidate) => candidate.path === OUTCOME_FIELD);
    const text = typeof outcome?.after === "string" ? outcome.after : "";
    rows.push({
      key: recordName(change),
      before: valueText(field.before, field.beforeRef, t),
      after: valueText(field.after, field.afterRef, t),
      refused: text.startsWith(REFUSED_PREFIX) ? text.slice(REFUSED_PREFIX.length).trim() : null,
    });
  }
  // Refused records lead, so the ones that need a look are the ones on screen.
  return rows.sort((a, b) => Number(b.refused !== null) - Number(a.refused !== null));
}

/** What the dry run says would happen, in one line, and whether it needs a look. */
export function dryRun(
  preview: ProposalPreviewData,
  t: TranslateFn,
): { text: string; warn: boolean } {
  const facts = approvalFacts(preview, t);
  if (facts.wouldFail) {
    return { text: facts.wouldFail, warn: true };
  }
  if (facts.refused.length > 0) {
    const [reason] = refusalReasons(facts.refused)[0] ?? [""];
    return {
      text: t(
        "{0} would go through. {1} would be refused: {2}",
        facts.count - facts.refused.length,
        facts.refused.length,
        reason.charAt(0).toLowerCase() + reason.slice(1),
      ),
      warn: true,
    };
  }
  if (facts.changedSince > 0) {
    return {
      text: t(
        "{0, plural, one {# record changed since the agent drafted this.} other {# records changed since the agent drafted this.}}",
        facts.changedSince,
      ),
      warn: true,
    };
  }
  if (facts.count > 1) {
    return {
      text: t("All {0} would go through as shown.", recordCount(facts.resource, facts.count, t)),
      warn: false,
    };
  }
  return { text: t("It would go through as shown."), warn: false };
}

export function Highlights({ rows }: { rows: { label: string; value: string }[] }) {
  if (rows.length === 0) {
    return null;
  }
  return (
    <div className="dk-dc2-diff">
      {rows.map((row) => (
        <div key={row.label} className="dk-dc2-dr dk-dc2-kv">
          <span className="dk-dc2-dk">{row.label}</span>
          <b>{row.value}</b>
        </div>
      ))}
    </div>
  );
}

/**
 * A plan, decided as one: every step runs in order, or none does. It is
 * never decided together with other changes, so it keeps a card of its own.
 */
export function PlanCard({ node, leaving, onGate, onDeclineWith }: CardProps) {
  const t = useT();
  const stepsQuery = usePlanSteps(node.id);
  const previewQuery = usePlanPreview({ scope: "approver", id: node.id });
  const approval = useApprovalGate(previewQuery);
  const stepsById = useMemo(
    () => new Map((stepsQuery.data ?? []).map((step) => [step.id, step])),
    [stepsQuery.data],
  );

  const reviewRequired = reviewRequiredFor(
    previewQuery.data,
    stepsQuery.data?.some((step) => step.reviewRequired) ?? true,
    false,
  );
  const approvable = canApprove(approval.gate, reviewRequired);
  const digest = gateDigest(approval.gate);
  const { handleDecisionError } = approval;
  useEffect(() => {
    onGate({
      approvable,
      digest,
      refused: 0,
      count: node.stepCount,
      handleError: handleDecisionError,
    });
  }, [approvable, digest, handleDecisionError, node.stepCount, onGate]);

  const wouldFail: WouldFailActions = {
    onAskAgent: (reasons) => onDeclineWith(askAgentPlanMessage(node.title, reasons, t)),
  };

  return (
    <QueueCard leaving={leaving} className="flex flex-col gap-4.5">
      <div className="flex flex-col gap-2.5">
        <Who node={node} />
        <h1>
          {node.title}
          <span> · {t("{0, plural, one {# step} other {# steps}}", node.stepCount)}</span>
        </h1>
        {node.summary !== "" && (
          <p className="max-w-[60ch] text-base leading-relaxed break-words text-dsk-fg2">
            {node.summary}
          </p>
        )}
      </div>
      <div className="overflow-hidden rounded-xl bg-dsk-raised p-4 shadow-(--dsk-lift)">
        <PreviewLoadState
          query={previewQuery}
          changed={approval.changed}
          reviewRequired={reviewRequired}
        >
          {(plan) => (
            <PlanPreview
              plan={plan}
              stepTitle={(proposalId) => {
                const step = stepsById.get(proposalId);
                return step ? presentProposal(asAssistantProposal(step)).summary : null;
              }}
              wouldFail={wouldFail}
            />
          )}
        </PreviewLoadState>
      </div>
      <p className="flex items-start gap-2 text-dsk-base text-dsk-subtle">
        <span className="mt-0.5 shrink-0">
          <DeskIcon name="info" size={13} stroke={2} />
        </span>
        {t("Decided as one: every step runs in order, or none does.")}
      </p>
    </QueueCard>
  );
}
