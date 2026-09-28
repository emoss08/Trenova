import { AgentTile } from "@/components/agent-identity/agent-tile";
import { conversationPath } from "@/lib/conversation-path";
import { SimulationLine } from "@/components/assistant/proposal-card";
import type { EditorFocus } from "@/components/assistant/proposal-editor";
import { PlanPreview } from "@/components/assistant/proposal-preview/plan-preview";
import { canApprove } from "@/components/assistant/proposal-preview/preview-gate";
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
import { argumentRows } from "@/components/assistant/proposal-state";
import { toneVar } from "@/components/kpi/tone";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { CheckIcon, ChevronRightIcon, PencilIcon, TriangleAlertIcon, XIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { Link } from "react-router";
import { asAssistantProposal } from "./decision-presenters";
import {
  isPendingPlan,
  isPendingProposal,
  usePlanSteps,
  type PendingDecisionNode,
  type PendingPlanNode,
  type PendingProposalNode,
  type PlanStepNode,
} from "./use-pending-decisions";
import { agentRunPath } from "@/lib/record-paths";

export type DecisionActions = {
  onAccept: () => void;
  /** Rejects; a reason given here is what the rejection opens with. */
  onReject: (initialReason?: string) => void;
  /** Opens the editor, on one value when a reason named it. */
  onModify?: (focus?: EditorFocus) => void;
  busy: boolean;
};

/**
 * The focused row read whole: what it would change, why the agent wants it,
 * what it would have done in simulation, and the run and conversation it
 * came from. The decision buttons live here and answer the same keys as
 * the list.
 *
 * `onPreviewShown` hears the digest of every proposal preview this pane puts
 * on screen, so a batch approval can say which of its proposals the person
 * actually reviewed.
 */
export function DecisionDetail({
  node,
  actions,
  onPreviewShown,
}: {
  node: PendingDecisionNode | null;
  actions: DecisionActions;
  onPreviewShown?: (proposalId: string, digest: string) => void;
}) {
  const t = useT();

  if (node === null) {
    return (
      <div className="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center text-sm">
        <p>{t("Pick a row to read it here.")}</p>
        <p className="flex items-center gap-1.5 text-xs">
          <Kbd>j</Kbd>
          <Kbd>k</Kbd>
          {t("move")} · <Kbd>a</Kbd> {t("approve")} · <Kbd>r</Kbd> {t("reject")} · <Kbd>m</Kbd>{" "}
          {t("modify")} · <Kbd>x</Kbd> {t("mark")}
        </p>
      </div>
    );
  }

  if (isPendingProposal(node)) {
    return <ProposalDetail node={node} actions={actions} onPreviewShown={onPreviewShown} />;
  }
  if (isPendingPlan(node)) {
    return <PlanDetail node={node} actions={actions} />;
  }

  return null;
}

function Provenance({ node }: { node: PendingDecisionNode }) {
  const t = useT();
  const run = node.run;
  const definition = run?.definition ?? null;
  const conversationId = run?.subjectType === "AssistantThread" ? run.subjectId : null;

  return (
    <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
      <span className="flex items-center gap-1.5">
        <AgentTile agent={definition} size="xs" />
        {definition?.name ?? t("Retired agent")}
      </span>
      <span>{formatUnixDateTimeMedium(node.createdAt)}</span>
      {conversationId && (
        <Link to={conversationPath(conversationId)} className="text-brand hover:underline">
          {t("Open the conversation")}
        </Link>
      )}
      {run && (
        <Link to={agentRunPath(run.id)} className="text-brand hover:underline">
          {t("Open the run")}
        </Link>
      )}
    </div>
  );
}

function ActionRow({
  actions,
  reversible,
  editable,
  approvable,
}: {
  actions: DecisionActions;
  reversible: boolean;
  editable: boolean;
  /** False while the preview is first read and once its record has moved. */
  approvable: boolean;
}) {
  const t = useT();

  return (
    <div className="border-border flex items-center gap-2 border-t pt-3">
      <Button size="sm" onClick={actions.onAccept} disabled={actions.busy || !approvable}>
        <CheckIcon className="size-3.5" />
        {t("Approve")}
        <Kbd className="ml-1">a</Kbd>
      </Button>
      <Button
        size="sm"
        variant="outline"
        onClick={() => actions.onReject()}
        disabled={actions.busy}
      >
        <XIcon className="size-3.5" />
        {t("Reject")}
        <Kbd className="ml-1">r</Kbd>
      </Button>
      {editable && actions.onModify && (
        <Button
          size="sm"
          variant="ghost"
          onClick={() => actions.onModify?.()}
          disabled={actions.busy || !approvable}
        >
          <PencilIcon className="size-3.5" />
          {t("Modify")}
          <Kbd className="ml-1">m</Kbd>
        </Button>
      )}
      {!reversible && (
        <span
          className="ml-auto flex items-center gap-1 text-xs"
          style={{ color: toneVar("warning") }}
        >
          <TriangleAlertIcon className="size-3" />
          {t("Permanent")}
        </span>
      )}
    </div>
  );
}

function Highlights({ highlights }: { highlights: { label: string; value: string }[] }) {
  if (highlights.length === 0) {
    return null;
  }

  return (
    <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-sm">
      {highlights.map((entry) => (
        <div key={entry.label} className="contents">
          <dt className="text-muted-foreground">{entry.label}</dt>
          <dd className="break-words">{entry.value}</dd>
        </div>
      ))}
    </dl>
  );
}

/**
 * The literal arguments the agent sent, for anyone who wants them. They are
 * what the preview was computed from, not what a person decides on, so they
 * sit behind a disclosure below it.
 */
function ArgumentDetails({ rows }: { rows: { key: string; value: string }[] }) {
  const t = useT();
  if (rows.length === 0) {
    return null;
  }

  return (
    <Collapsible className="flex flex-col gap-1">
      <CollapsibleTrigger className="text-muted-foreground ui-focus-ring group flex items-center gap-1 self-start rounded-md text-xs font-medium">
        <ChevronRightIcon className="size-3 transition-transform group-data-[panel-open]:rotate-90" />
        {t("Details")}
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="flex flex-col gap-1 pt-1">
          <span className="text-muted-foreground text-xs">{t("Exactly as proposed")}</span>
          <div className="flex flex-wrap gap-1">
            {rows.map((row) => (
              <span
                key={row.key}
                className="bg-sunken inline-flex max-w-full items-center gap-1 rounded-md px-1.5 py-0.5 text-xs"
              >
                <span className="text-muted-foreground font-mono">{row.key}</span>
                <span className="truncate">{row.value}</span>
              </span>
            ))}
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function ProposalDetail({
  node,
  actions,
  onPreviewShown,
}: {
  node: PendingProposalNode;
  actions: DecisionActions;
  onPreviewShown?: (proposalId: string, digest: string) => void;
}) {
  const t = useT();
  const proposal = useMemo(() => asAssistantProposal(node), [node]);
  const view = useMemo(() => presentProposal(proposal), [proposal]);
  const rows = useMemo(() => argumentRows(proposal.arguments ?? null), [proposal.arguments]);
  const confidence = Math.round(Math.min(1, Math.max(0, node.confidence)) * 100);
  const previewQuery = useProposalPreview({ scope: "approver", id: node.id });
  const approval = useApprovalGate(previewQuery);
  const shownDigest = previewQuery.data?.digest;

  useEffect(() => {
    if (shownDigest !== undefined) {
      onPreviewShown?.(node.id, shownDigest);
    }
  }, [node.id, onPreviewShown, shownDigest]);

  // A write that would be refused: change the value it names here, or turn
  // it down with the reasons so the agent in its conversation learns what to
  // fix. Approve stays offered and warned, as the chat card leaves it.
  const editable = proposal.fields.length > 0 && actions.onModify !== undefined;
  const approvable = canApprove(approval.gate);
  const wouldFail: WouldFailActions = {
    canChange: editable ? (param) => editableParam(param, proposal.fields) : undefined,
    onChange:
      editable && approvable && !actions.busy
        ? (reason) => actions.onModify?.({ param: reason.param, label: reason.label })
        : undefined,
    onAskAgent: actions.busy
      ? undefined
      : (reasons) => actions.onReject(askAgentMessage(proposal.toolName, reasons, t)),
  };

  return (
    <article className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-5">
      <header className="flex flex-col gap-1.5">
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
            {view.title}
          </span>
          {view.severity && (
            <span
              className="shrink-0 text-xs font-medium"
              style={{ color: toneVar(view.severity.tone) }}
            >
              {view.severity.label}
            </span>
          )}
          <Badge variant="neutral" className="h-4 shrink-0 px-1.5 text-2xs tabular-nums">
            {t("{0}% sure", confidence)}
          </Badge>
        </div>
        <h2 className="text-base leading-snug font-semibold">{view.summary}</h2>
        <Provenance node={node} />
      </header>

      <section className="flex flex-col gap-2">
        <h3 className="text-muted-foreground text-xs font-medium">{t("What changes")}</h3>
        <PreviewLoadState
          query={previewQuery}
          changed={approval.changed}
          fallback={<Highlights highlights={view.highlights} />}
        >
          {(preview) => <ProposalPreview preview={preview} wouldFail={wouldFail} />}
        </PreviewLoadState>
      </section>

      {node.rationale !== "" && (
        <section className="flex flex-col gap-1">
          <h3 className="text-muted-foreground text-xs font-medium">{t("Why")}</h3>
          <p className="text-sm leading-relaxed whitespace-pre-wrap">{node.rationale}</p>
        </section>
      )}

      {proposal.simulation && (
        <section className="flex flex-col gap-1">
          <h3 className="text-muted-foreground text-xs font-medium">
            {t("What it would have done")}
          </h3>
          <SimulationLine simulation={proposal.simulation} />
        </section>
      )}

      {node.evidence.length > 0 && (
        <section className="flex flex-col gap-1">
          <h3 className="text-muted-foreground text-xs font-medium">{t("Evidence")}</h3>
          <ul className="flex flex-col gap-1 text-xs">
            {node.evidence.map((item) => (
              <li key={`${item.type}-${item.id}`} className="flex gap-2">
                <span className="text-muted-foreground shrink-0">{item.type}</span>
                <span className="min-w-0 flex-1">{item.note || item.id}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <ArgumentDetails rows={rows} />

      <ActionRow
        actions={actions}
        reversible={view.reversible}
        editable={proposal.fields.length > 0}
        approvable={approvable}
      />
    </article>
  );
}

/** A plan step's sentence, the same one the chat card and the queue row say. */
function StepSentence({ step }: { step: PlanStepNode | undefined }) {
  const t = useT();
  if (!step) {
    return <span>{t("A step no longer in the plan")}</span>;
  }
  const view = presentProposal(asAssistantProposal(step));

  return (
    <span className="flex flex-col gap-0.5">
      <span>{view.summary}</span>
      {step.rationale !== "" && (
        <span className="text-muted-foreground text-xs">{step.rationale}</span>
      )}
    </span>
  );
}

function PlanDetail({ node, actions }: { node: PendingPlanNode; actions: DecisionActions }) {
  const t = useT();
  const stepsQuery = usePlanSteps(node.id);
  const previewQuery = usePlanPreview({ scope: "approver", id: node.id });
  const approval = useApprovalGate(previewQuery);
  const stepsById = useMemo(
    () => new Map((stepsQuery.data ?? []).map((step) => [step.id, step])),
    [stepsQuery.data],
  );
  // A plan is decided whole, so a step that would be refused can only be
  // turned down with its reasons.
  const wouldFail: WouldFailActions = {
    onAskAgent: actions.busy
      ? undefined
      : (reasons) => actions.onReject(askAgentPlanMessage(node.title, reasons, t)),
  };

  return (
    <article className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-5">
      <header className="flex flex-col gap-1.5">
        <span className="text-muted-foreground text-xs">
          {t(
            "{0, plural, one {A plan of # step} other {A plan of # steps}}, decided as one",
            node.stepCount,
          )}
        </span>
        <h2 className="text-base leading-snug font-semibold">{node.title}</h2>
        {node.summary !== "" && (
          <p className="text-muted-foreground text-sm leading-relaxed">{node.summary}</p>
        )}
        <Provenance node={node} />
      </header>

      <section className="flex flex-col gap-2">
        <h3 className="text-muted-foreground text-xs font-medium">
          {t("What changes, in the order it runs")}
        </h3>
        {stepsQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
          </div>
        ) : (
          <PreviewLoadState
            query={previewQuery}
            changed={approval.changed}
            fallback={<StepSentences steps={stepsQuery.data ?? []} />}
          >
            {(plan) => (
              <PlanPreview
                plan={plan}
                stepTitle={(proposalId) => <StepSentence step={stepsById.get(proposalId)} />}
                wouldFail={wouldFail}
              />
            )}
          </PreviewLoadState>
        )}
      </section>

      <ActionRow
        actions={actions}
        reversible
        editable={false}
        approvable={canApprove(approval.gate)}
      />
    </article>
  );
}

/** The steps as sentences alone, for when their preview could not be read. */
function StepSentences({ steps }: { steps: PlanStepNode[] }) {
  return (
    <ol className="flex flex-col gap-2">
      {steps.map((step) => (
        <li key={step.id} className="flex gap-2 text-sm">
          <span className="text-muted-foreground w-5 shrink-0 text-right tabular-nums">
            {step.planStep}.
          </span>
          <span className="min-w-0 flex-1">
            <StepSentence step={step} />
          </span>
        </li>
      ))}
    </ol>
  );
}
