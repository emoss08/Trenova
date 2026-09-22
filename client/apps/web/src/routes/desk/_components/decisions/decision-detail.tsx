import { AgentTile } from "@/components/agent-identity/agent-tile";
import { SimulationLine } from "@/components/assistant/proposal-card";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { argumentRows } from "@/components/assistant/proposal-state";
import { toneVar } from "@/components/kpi/tone";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { CheckIcon, PencilIcon, TriangleAlertIcon, XIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { asAssistantProposal } from "./decision-presenters";
import {
  isPendingPlan,
  isPendingProposal,
  usePlanSteps,
  type PendingDecisionNode,
  type PendingPlanNode,
  type PendingProposalNode,
} from "./use-pending-decisions";
import { agentRunPath } from "@/lib/record-paths";

export type DecisionActions = {
  onAccept: () => void;
  onReject: () => void;
  onModify?: () => void;
  busy: boolean;
};

/**
 * The focused row read whole: what would change, why the agent wants it,
 * what it would have done in simulation, and the run and conversation it
 * came from. The decision buttons live here and answer the same keys as
 * the list.
 */
export function DecisionDetail({
  node,
  actions,
}: {
  node: PendingDecisionNode | null;
  actions: DecisionActions;
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
    return <ProposalDetail node={node} actions={actions} />;
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
        <Link to={`/desk/t/${conversationId}`} className="text-brand hover:underline">
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
}: {
  actions: DecisionActions;
  reversible: boolean;
  editable: boolean;
}) {
  const t = useT();

  return (
    <div className="border-border flex items-center gap-2 border-t pt-3">
      <Button size="sm" onClick={actions.onAccept} disabled={actions.busy}>
        <CheckIcon className="size-3.5" />
        {t("Approve")}
        <Kbd className="ml-1">a</Kbd>
      </Button>
      <Button size="sm" variant="outline" onClick={actions.onReject} disabled={actions.busy}>
        <XIcon className="size-3.5" />
        {t("Reject")}
        <Kbd className="ml-1">r</Kbd>
      </Button>
      {editable && actions.onModify && (
        <Button size="sm" variant="ghost" onClick={actions.onModify} disabled={actions.busy}>
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

function ProposalDetail({
  node,
  actions,
}: {
  node: PendingProposalNode;
  actions: DecisionActions;
}) {
  const t = useT();
  const proposal = useMemo(() => asAssistantProposal(node), [node]);
  const view = useMemo(() => presentProposal(proposal), [proposal]);
  const rows = useMemo(() => argumentRows(proposal.arguments ?? null), [proposal.arguments]);
  const confidence = Math.round(Math.min(1, Math.max(0, node.confidence)) * 100);

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

      {view.highlights.length > 0 && (
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-sm">
          {view.highlights.map((entry) => (
            <div key={entry.label} className="contents">
              <dt className="text-muted-foreground">{entry.label}</dt>
              <dd className="break-words">{entry.value}</dd>
            </div>
          ))}
        </dl>
      )}

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

      {rows.length > 0 && (
        <section className="flex flex-col gap-1">
          <h3 className="text-muted-foreground text-xs font-medium">{t("Exactly as proposed")}</h3>
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

      <ActionRow
        actions={actions}
        reversible={view.reversible}
        editable={proposal.fields.length > 0}
      />
    </article>
  );
}

function PlanDetail({ node, actions }: { node: PendingPlanNode; actions: DecisionActions }) {
  const t = useT();
  const stepsQuery = usePlanSteps(node.id);

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

      <section className="flex flex-col gap-1">
        <h3 className="text-muted-foreground text-xs font-medium">
          {t("Steps, in the order they run")}
        </h3>
        {stepsQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
          </div>
        ) : (
          <ol className="flex flex-col gap-2">
            {(stepsQuery.data ?? []).map((step) => {
              const view = presentProposal(asAssistantProposal(step));
              return (
                <li key={step.id} className="flex gap-2 text-sm">
                  <span className="text-muted-foreground w-5 shrink-0 text-right tabular-nums">
                    {step.planStep}.
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block">{view.summary}</span>
                    {step.rationale !== "" && (
                      <span className="text-muted-foreground block text-xs">{step.rationale}</span>
                    )}
                  </span>
                </li>
              );
            })}
          </ol>
        )}
      </section>

      <ActionRow actions={actions} reversible editable={false} />
    </article>
  );
}
