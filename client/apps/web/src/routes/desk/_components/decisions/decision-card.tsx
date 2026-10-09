import { PlanPreview } from "@/components/assistant/proposal-preview/plan-preview";
import { canApprove, gateDigest } from "@/components/assistant/proposal-preview/preview-gate";
import type { EditorFocus } from "@/components/assistant/proposal-editor";
import {
  askAgentMessage,
  askAgentPlanMessage,
  editableParam,
} from "@/components/assistant/proposal-preview/preview-warnings";
import {
  PreviewLoadState,
  ProposalPreview,
  type WouldFailActions,
} from "@/components/assistant/proposal-preview/proposal-preview";
import {
  useApprovalGate,
  usePlanPreview,
  useProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { conversationPath } from "@/lib/conversation-path";
import type { ProposalPreview as ProposalPreviewData } from "@/lib/graphql/agent-preview";
import { agentRunPath } from "@/lib/record-paths";
import { formatTimeAgo } from "@/lib/time-utils";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router";
import {
  approvalFacts,
  OUTCOME_FIELD,
  recordCount,
  refusalReasons,
  REFUSED_PREFIX,
  valueText,
} from "@/components/desk-chat/conversation/approval-facts";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { asAssistantProposal } from "./decision-presenters";
import {
  isPendingPlan,
  usePlanSteps,
  type PendingDecisionNode,
  type PendingPlanNode,
  type PendingProposalNode,
} from "./use-pending-decisions";
import { useRichT } from "@trenova/shared/i18n/rich";

/** The most record rows the card lists before it says how many more there are. */
const SHOWN_ROWS = 5;

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
  node: PendingDecisionNode;
  leaving: "ok" | "no" | null;
  /** The card underneath: the like-batch line and anything else the queue adds. */
  footer?: ReactNode;
  onGate: (gate: CardGate) => void;
  onPreviewShown: (proposalId: string, digest: string) => void;
  /** Turns the change down with a reason already written. */
  onDeclineWith: (reason: string) => void;
  /** Opens the editor, on one value when a refusal named it; absent when values can't be changed. */
  onModify?: (focus?: EditorFocus) => void;
};

/**
 * One decision, read whole: who wants it and when, what it changes record by
 * record, what the dry run says would happen, and the exact arguments behind
 * it for anyone who wants them.
 */
export function DecisionCard(props: CardProps) {
  return isPendingPlan(props.node) ? (
    <PlanCard {...props} node={props.node} />
  ) : (
    <ProposalCard {...props} node={props.node as PendingProposalNode} />
  );
}

function Who({ node }: { node: PendingDecisionNode }) {
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

function RawArguments({ value }: { value: unknown }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        variant="bare"
        size="bare"
        className="mt-4 gap-1.5 text-sm text-dsk-subtle hover:text-dsk-fg"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <span
          aria-hidden
          style={{ display: "inline-flex", transform: open ? "rotate(90deg)" : undefined }}
        >
          <DeskIcon name="chevR" size={10} stroke={2.4} />
        </span>
        {t("Arguments the agent sent")}
      </Button>
      {open && <pre className="dk-dc2-pre">{JSON.stringify(value, null, 2)}</pre>}
    </>
  );
}

type DiffRow = { key: string; before: string; after: string; refused: string | null };

/**
 * The records a write touches, one line each with the value it sets, when
 * every record has the same field set; null when the full preview should
 * draw it instead.
 */
function diffRows(preview: ProposalPreviewData, t: TranslateFn): DiffRow[] | null {
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
      key: change.label || change.entityId || "",
      before: valueText(field.before, field.beforeRef, t),
      after: valueText(field.after, field.afterRef, t),
      refused: text.startsWith(REFUSED_PREFIX) ? text.slice(REFUSED_PREFIX.length).trim() : null,
    });
  }
  // Refused records lead, so the ones that need a look are the ones on screen.
  return rows.sort((a, b) => Number(b.refused !== null) - Number(a.refused !== null));
}

/** What the dry run says would happen, in one line, and whether it needs a look. */
function dryRun(preview: ProposalPreviewData, t: TranslateFn): { text: string; warn: boolean } {
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

function ProposalCard({
  node,
  leaving,
  footer,
  onGate,
  onPreviewShown,
  onDeclineWith,
  onModify,
}: CardProps & { node: PendingProposalNode }) {
  const t = useT();
  const proposal = useMemo(() => asAssistantProposal(node), [node]);
  const view = useMemo(() => presentProposal(proposal), [proposal]);
  const previewQuery = useProposalPreview({ scope: "approver", id: node.id });
  const approval = useApprovalGate(previewQuery);
  const preview = previewQuery.data;
  const facts = useMemo(() => approvalFacts(preview, t), [preview, t]);
  const rows = useMemo(() => (preview ? diffRows(preview, t) : null), [preview, t]);

  const approvable = canApprove(approval.gate);
  const digest = gateDigest(approval.gate);
  const { handleDecisionError } = approval;
  useEffect(() => {
    onGate({
      approvable,
      digest,
      refused: facts.refused.length,
      count: facts.count,
      handleError: handleDecisionError,
    });
  }, [approvable, digest, facts.count, facts.refused.length, handleDecisionError, onGate]);
  useEffect(() => {
    if (preview?.digest !== undefined) {
      onPreviewShown(node.id, preview.digest);
    }
  }, [node.id, onPreviewShown, preview?.digest]);

  // A write that would be refused: change the value a reason names, or turn
  // it down with the reasons so the agent learns what to fix.
  const editable = proposal.fields.length > 0 && onModify !== undefined;
  const wouldFail: WouldFailActions = {
    canChange: editable ? (param) => editableParam(param, proposal.fields) : undefined,
    onChange:
      editable && approvable
        ? (reason) => onModify?.({ param: reason.param, label: reason.label })
        : undefined,
    onAskAgent: (reasons) => onDeclineWith(askAgentMessage(proposal.toolName, reasons, t)),
  };
  const run = preview ? dryRun(preview, t) : null;

  return (
    <div className={cn("dk-dc2-card", leaving && `dk-out-${leaving}`)}>
      <Who node={node} />
      <h1>
        {view.title}
        {facts.count > 0 && <span> · {recordCount(facts.resource, facts.count, t)}</span>}
      </h1>
      <p className="dk-dc2-why">{node.rationale || view.summary}</p>

      <PreviewLoadState
        query={previewQuery}
        changed={approval.changed}
        fallback={<Highlights rows={view.highlights} />}
      >
        {(data) =>
          rows && rows.length > 0 ? (
            <div className="dk-dc2-diff">
              <div className="dk-dc2-dh">
                <span>
                  {data.changes
                    .find((change) => !change.withheld)
                    ?.fields.find((field) => !field.withheld && field.path !== OUTCOME_FIELD)
                    ?.label ?? ""}
                </span>
                <span className={cn("dk-dc2-rev", !view.reversible && "dk-hard")}>
                  {view.reversible ? t("Reversible") : t("Can't be undone")}
                </span>
              </div>
              {rows.slice(0, SHOWN_ROWS).map((row, index) => (
                <div
                  key={`${row.key}-${index}`}
                  className={cn("dk-dc2-dr", row.refused && "dk-bad")}
                >
                  <span className="dk-dc2-dk">{row.key}</span>
                  <s>{row.before}</s>
                  <i aria-hidden>→</i>
                  <span className="sr-only">{t("to")}</span>
                  <b title={row.refused ?? undefined}>
                    {row.refused ? t("Refused") : row.after}
                    {row.refused && <span className="sr-only">: {row.refused}</span>}
                  </b>
                </div>
              ))}
              {facts.count > Math.min(rows.length, SHOWN_ROWS) && (
                <div className="dk-dc2-more">
                  {t("+ {0} more like these", facts.count - Math.min(rows.length, SHOWN_ROWS))}
                </div>
              )}
            </div>
          ) : (
            <div className="dk-dc2-full">
              <ProposalPreview preview={data} wouldFail={wouldFail} />
            </div>
          )
        }
      </PreviewLoadState>

      {run && (
        <div className={cn("dk-dc2-sim", run.warn && "dk-warn")}>
          <span className="dk-dc2-simh">
            <DeskIcon name={run.warn ? "alert" : "shield"} size={12} stroke={2} />
            {t("Dry run")}
          </span>
          {run.text}
        </div>
      )}

      <RawArguments value={{ tool: node.toolName, ...proposal.arguments }} />
      {footer}
    </div>
  );
}

function Highlights({ rows }: { rows: { label: string; value: string }[] }) {
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

function PlanCard({
  node,
  leaving,
  footer,
  onGate,
  onDeclineWith,
}: CardProps & { node: PendingPlanNode }) {
  const t = useT();
  const stepsQuery = usePlanSteps(node.id);
  const previewQuery = usePlanPreview({ scope: "approver", id: node.id });
  const approval = useApprovalGate(previewQuery);
  const stepsById = useMemo(
    () => new Map((stepsQuery.data ?? []).map((step) => [step.id, step])),
    [stepsQuery.data],
  );

  const approvable = canApprove(approval.gate);
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
    <div className={cn("dk-dc2-card", leaving && `dk-out-${leaving}`)}>
      <Who node={node} />
      <h1>
        {node.title}
        <span> · {t("{0, plural, one {# step} other {# steps}}", node.stepCount)}</span>
      </h1>
      {node.summary !== "" && <p className="dk-dc2-why">{node.summary}</p>}
      <div className="dk-dc2-full">
        <PreviewLoadState query={previewQuery} changed={approval.changed}>
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
      <div className="dk-dc2-sim">
        <span className="dk-dc2-simh">
          <DeskIcon name="info" size={12} stroke={2} />
          {t("Plan")}
        </span>
        {t("Decided as one: every step runs in order, or none does.")}
      </div>
      {footer}
    </div>
  );
}
