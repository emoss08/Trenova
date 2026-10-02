import { proposalLabel } from "@/components/assistant/decision-chrome";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import { groupThread, type ThreadEntry, type ToolExchange } from "@/components/assistant/thread-view";
import { useThreadHistory } from "@/components/assistant/use-thread-history";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskStore, type DeskWorkspaceTab } from "@/stores/desk-store";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import {
  CheckIcon,
  CircleAlertIcon,
  CircleDashedIcon,
  HourglassIcon,
  MinusIcon,
  PanelRightCloseIcon,
  XIcon,
} from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useMemo, type ReactNode } from "react";
import { EASE_SETTLE } from "@/lib/motion";
import { ArtifactsPane } from "./artifacts/artifacts-pane";
import type { LiveArtifacts } from "./desk-layout";

/** The keystroke that folds the workspace, shown beside its control. */
export const WORKSPACE_SHORTCUT = "⌘\\";

export type DeskWorkspaceProps = {
  threadId: string;
  agent: AgentChoice | null;
  liveArtifacts: LiveArtifacts;
  onClose: () => void;
  className?: string;
};

/**
 * The workspace beside a conversation: what it produced, what it asked for,
 * and what it did.
 *
 * Three tabs over one frame. Artifacts is where a table or a draft is read
 * whole; Decisions lists every write the conversation has proposed with what
 * became of it, and opens the one still waiting; Activity is the
 * conversation's work in order, every call with what it was about and how
 * long it took, for a person who wants to check the agent's working rather
 * than take its word. The tab is remembered.
 */
export function DeskWorkspace({ threadId, agent, liveArtifacts, onClose, className }: DeskWorkspaceProps) {
  const t = useT();
  const tab = useDeskStore((state) => state.workspaceTab);
  const setTab = useDeskStore((state) => state.setWorkspaceTab);
  const reduceMotion = useReducedMotion();

  const artifactsQuery = useQuery(queries.assistant.artifacts(threadId));
  const proposalsQuery = useQuery(queries.assistant.proposals(threadId));
  const plansQuery = useQuery(queries.assistant.plans(threadId));
  const artifactCount = artifactsQuery.data?.results.length ?? 0;
  const proposals = useMemo(() => proposalsQuery.data?.results ?? [], [proposalsQuery.data]);
  const plans = useMemo(() => plansQuery.data?.results ?? [], [plansQuery.data]);
  const waiting =
    proposals.filter((proposal) => proposal.status === "Pending" && !proposal.planId).length +
    plans.filter((plan) => plan.status === "Pending").length;
  const decisionCount = plans.length + proposals.filter((proposal) => !proposal.planId).length;

  const tabs: { value: DeskWorkspaceTab; label: string; count: number; attention?: boolean }[] = [
    { value: "artifacts", label: t("Artifacts"), count: artifactCount },
    { value: "decisions", label: t("Decisions"), count: decisionCount, attention: waiting > 0 },
    { value: "activity", label: t("Activity"), count: 0 },
  ];

  return (
    <section
      data-slot="desk-workspace"
      aria-label={t("Workspace")}
      className={cn("bg-card flex h-full min-h-0 min-w-0 flex-col", className)}
    >
      <header className="border-desk-hairline flex h-11 shrink-0 items-center gap-1 border-b pr-1.5 pl-2">
        <div role="tablist" aria-label={t("Workspace")} className="flex min-w-0 items-center gap-0.5">
          {tabs.map((item) => {
            const selected = item.value === tab;

            return (
              <button
                key={item.value}
                type="button"
                role="tab"
                aria-selected={selected}
                onClick={() => setTab(item.value)}
                className={cn(
                  "ui-focus-ring relative flex h-7 items-center gap-1.5 rounded-md px-2.5 text-sm transition-colors",
                  selected
                    ? "text-foreground font-medium"
                    : "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
                )}
              >
                {selected && (
                  <m.span
                    layoutId={`desk-workspace-tab-${threadId}`}
                    aria-hidden
                    transition={
                      reduceMotion ? { duration: 0 } : { duration: 0.24, ease: EASE_SETTLE }
                    }
                    className="bg-surface-selected absolute inset-0 rounded-md"
                  />
                )}
                <span className="relative">{item.label}</span>
                {item.count > 0 && (
                  <span
                    className={cn(
                      "relative text-xs tabular-nums",
                      item.attention ? "text-warning font-medium" : "text-foreground-subtle",
                    )}
                  >
                    {item.count}
                  </span>
                )}
              </button>
            );
          })}
        </div>
        <span className="flex-1" />
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-foreground-subtle hover:text-foreground shrink-0"
                aria-label={t("Hide the workspace")}
                onClick={onClose}
              />
            }
          >
            <PanelRightCloseIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent side="bottom" className="flex items-center gap-2">
            {t("Hide the workspace")}
            <Kbd>{WORKSPACE_SHORTCUT}</Kbd>
          </TooltipContent>
        </Tooltip>
      </header>

      <div role="tabpanel" className="flex min-h-0 min-w-0 flex-1 flex-col">
        {tab === "artifacts" && (
          <ArtifactsPane
            key={threadId}
            threadId={threadId}
            liveArtifacts={liveArtifacts}
            onClose={onClose}
            framed
            className="min-h-0 flex-1"
          />
        )}
        {tab === "decisions" && (
          <DecisionsTab
            threadId={threadId}
            agent={agent}
            proposals={proposals}
            plans={plans}
            isLoading={proposalsQuery.isLoading || plansQuery.isLoading}
          />
        )}
        {tab === "activity" && <ActivityTab threadId={threadId} agent={agent} />}
      </div>
    </section>
  );
}

// ------------------------------------------------------------ decisions

type DecisionItem =
  | { kind: "proposal"; proposal: AssistantProposal; at: number }
  | { kind: "plan"; plan: AssistantPlan; steps: AssistantProposal[]; at: number };

function decisionItems(proposals: AssistantProposal[], plans: AssistantPlan[]): DecisionItem[] {
  const items: DecisionItem[] = [];
  for (const plan of plans) {
    items.push({
      kind: "plan",
      plan,
      steps: proposals
        .filter((proposal) => proposal.planId === plan.id)
        .sort((a, b) => a.planStep - b.planStep),
      at: plan.createdAt,
    });
  }
  for (const proposal of proposals) {
    if (!proposal.planId) {
      items.push({ kind: "proposal", proposal, at: proposal.createdAt });
    }
  }

  return items.sort((a, b) => b.at - a.at);
}

function statusTone(status: string): { icon: typeof CheckIcon; className: string; label: (t: TranslateFn) => string } {
  switch (status) {
    case "Executed":
    case "Completed":
    case "Accepted":
    case "Modified":
    case "Approved":
      return { icon: CheckIcon, className: "text-success", label: (t) => t("Approved") };
    case "ExecutionFailed":
    case "Failed":
      return { icon: CircleAlertIcon, className: "text-danger", label: (t) => t("Failed") };
    case "Rejected":
      return { icon: XIcon, className: "text-foreground-muted", label: (t) => t("Rejected") };
    case "Skipped":
    case "Expired":
    case "Superseded":
      return { icon: MinusIcon, className: "text-foreground-subtle", label: (t) => t("Skipped") };
    case "Pending":
      return { icon: HourglassIcon, className: "text-warning", label: (t) => t("Waiting") };
    default:
      return { icon: CircleDashedIcon, className: "text-foreground-subtle", label: () => status };
  }
}

function DecisionsTab({
  threadId,
  agent,
  proposals,
  plans,
  isLoading,
}: {
  threadId: string;
  agent: AgentChoice | null;
  proposals: AssistantProposal[];
  plans: AssistantPlan[];
  isLoading: boolean;
}) {
  const t = useT();
  const focusDecision = useAssistantStore((state) => state.focusDecision);
  const items = useMemo(() => decisionItems(proposals, plans), [proposals, plans]);

  if (isLoading) {
    return <WorkspaceSkeleton />;
  }
  if (items.length === 0) {
    return (
      <WorkspaceEmpty title={t("No decisions yet")}>
        {t("A change the agent proposes waits here for you, and what became of it stays.")}
      </WorkspaceEmpty>
    );
  }

  return (
    <ScrollArea className="min-h-0 flex-1" maskHeight={12}>
      <ol className="flex flex-col gap-2 p-3">
        {items.map((item, index) => {
          const status = item.kind === "plan" ? item.plan.status : item.proposal.status;
          const tone = statusTone(status);
          const Icon = tone.icon;
          const pending = status === "Pending";
          const title =
            item.kind === "plan"
              ? item.plan.title
              : proposalLabel(item.proposal.toolName, item.proposal.arguments, t);
          const agentName = item.kind === "plan" ? item.plan.agentName : item.proposal.agentName;
          const decidedAt = item.kind === "plan" ? item.plan.decidedAt : item.proposal.decidedAt;

          return (
            <li
              key={item.kind === "plan" ? item.plan.id : item.proposal.id}
              style={{ animationDelay: `${Math.min(index, 8) * 30}ms` }}
              className="animate-rise ring-foreground/10 flex flex-col gap-2 rounded-lg p-3 ring-1"
            >
              <div className="flex items-start gap-2.5">
                <span className={cn("mt-0.5 flex size-4 shrink-0 items-center justify-center", tone.className)}>
                  <Icon className="size-3.5" />
                </span>
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <p className="truncate text-sm font-medium">{title}</p>
                  <p className="text-foreground-subtle flex min-w-0 items-center gap-1.5 text-xs">
                    <span className={cn(pending && "text-warning")}>{tone.label(t)}</span>
                    {decidedAt ? (
                      <>
                        <span aria-hidden>·</span>
                        <time dateTime={new Date(decidedAt * 1000).toISOString()} title={formatUnixDateTimeMedium(decidedAt)}>
                          {formatRelativeTime(Math.min(0, decidedAt - Math.floor(Date.now() / 1000)))}
                        </time>
                      </>
                    ) : null}
                    {agentName && (
                      <>
                        <span aria-hidden>·</span>
                        <span className="truncate">{agentName}</span>
                      </>
                    )}
                  </p>
                </div>
                {pending && (
                  <Button
                    size="xs"
                    variant="outline"
                    onClick={() =>
                      focusDecision(
                        threadId,
                        item.kind === "plan"
                          ? { proposalIds: [], planId: item.plan.id }
                          : { proposalIds: [item.proposal.id], planId: "" },
                        [item.kind === "plan" ? `plan:${item.plan.id}` : `proposal:${item.proposal.id}`],
                      )
                    }
                  >
                    {t("Decide")}
                  </Button>
                )}
              </div>
              {item.kind === "plan" && item.steps.length > 0 && (
                <ol className="border-border-subtle ml-1.5 flex flex-col gap-1 border-l pl-4">
                  {item.steps.map((step) => {
                    const stepTone = statusTone(step.status);
                    const StepIcon = stepTone.icon;

                    return (
                      <li key={step.id} className="text-foreground-muted flex items-center gap-2 text-xs">
                        <StepIcon className={cn("size-3 shrink-0", stepTone.className)} />
                        <span className="truncate">{proposalLabel(step.toolName, step.arguments, t)}</span>
                      </li>
                    );
                  })}
                </ol>
              )}
              {item.kind === "proposal" && item.proposal.executionError && (
                <p className="text-danger text-xs">{item.proposal.executionError}</p>
              )}
            </li>
          );
        })}
      </ol>
      {agent && <span className="sr-only">{agent.name}</span>}
    </ScrollArea>
  );
}

// ------------------------------------------------------------ activity

type ActivityRow = {
  key: string;
  at: number;
  exchange: ToolExchange;
};

type ActivityGroup = {
  key: string;
  entry: ThreadEntry & { kind: "assistant" };
  rows: ActivityRow[];
};

function activityGroups(entries: ThreadEntry[]): ActivityGroup[] {
  const groups: ActivityGroup[] = [];
  for (const entry of entries) {
    if (entry.kind !== "assistant" || entry.tools.length === 0) {
      continue;
    }
    groups.push({
      key: entry.message.id,
      entry,
      rows: entry.tools.map((exchange) => ({
        key: exchange.call.id,
        at: exchange.result?.createdAt ?? entry.message.createdAt,
        exchange,
      })),
    });
  }

  return groups.reverse();
}

function ActivityTab({ threadId, agent }: { threadId: string; agent: AgentChoice | null }) {
  const t = useT();
  const history = useThreadHistory(threadId);
  const groups = useMemo(() => activityGroups(groupThread(history.messages)), [history.messages]);

  if (history.isLoading) {
    return <WorkspaceSkeleton />;
  }
  if (groups.length === 0) {
    return (
      <WorkspaceEmpty title={t("Nothing looked up yet")}>
        {t("Every call the agent makes is listed here in order, with what it was about and how long it took.")}
      </WorkspaceEmpty>
    );
  }

  return (
    <ScrollArea className="min-h-0 flex-1" maskHeight={12}>
      <ol className="flex flex-col gap-4 p-3">
        {groups.map((group, index) => (
          <li
            key={group.key}
            style={{ animationDelay: `${Math.min(index, 8) * 30}ms` }}
            className="animate-rise flex flex-col gap-1.5"
          >
            <p className="text-foreground-subtle flex items-center gap-2 px-1 text-xs">
              <AgentTile agent={agent} size="xs" className="size-3.5" />
              <span className="truncate">{agent?.name ?? t("Assistant")}</span>
              <span aria-hidden>·</span>
              <time dateTime={new Date(group.entry.message.createdAt * 1000).toISOString()}>
                {formatUnixDateTimeMedium(group.entry.message.createdAt)}
              </time>
              {group.entry.message.latencyMs ? (
                <span className="ml-auto tabular-nums">
                  {formatWorkDuration(group.entry.message.latencyMs / 1000)}
                </span>
              ) : null}
            </p>
            <ol className="ring-foreground/10 divide-border-subtle flex flex-col divide-y rounded-lg ring-1">
              {group.rows.map((row) => {
                const description = describeToolCall(row.exchange.call.name, row.exchange.call.arguments);
                const failed = row.exchange.result?.toolFailed === true;

                return (
                  <li key={row.key} className="flex items-center gap-2.5 px-3 py-2 text-sm">
                    <span
                      className={cn(
                        "size-1.5 shrink-0 rounded-full",
                        failed ? "bg-danger" : "bg-foreground-subtle",
                      )}
                    />
                    <span className="min-w-0 flex-1 truncate">
                      <span className="text-foreground">{description.title}</span>
                      {description.subject && (
                        <span className="text-foreground-muted"> · {description.subject}</span>
                      )}
                    </span>
                    {row.exchange.result?.summary && (
                      <span className="text-foreground-subtle max-w-40 shrink-0 truncate text-xs">
                        {row.exchange.result.summary}
                      </span>
                    )}
                  </li>
                );
              })}
            </ol>
          </li>
        ))}
      </ol>
    </ScrollArea>
  );
}

// ------------------------------------------------------------ shared

function WorkspaceEmpty({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col items-center justify-center gap-1.5 px-8 text-center">
      <p className="text-sm font-medium">{title}</p>
      <p className="text-muted-foreground max-w-72 text-xs text-pretty">{children}</p>
    </div>
  );
}

function WorkspaceSkeleton() {
  return (
    <div className="flex flex-col gap-2 p-3" aria-busy>
      <Skeleton className="h-14" />
      <Skeleton className="h-14" />
      <Skeleton className="h-14 w-3/4" />
    </div>
  );
}
