import { AgentTile } from "@/components/agent-identity/agent-tile";
import { AskBox } from "@/components/assistant/ask-box";
import { queries } from "@/lib/queries";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AssistantThread } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowRightIcon, BotIcon, InboxIcon, PlugZapIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { BriefingPanel } from "./briefing-panel";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const RECENT_LIMIT = 5;

export type DeskHomeProps = {
  agents: AgentDefinitionRow[];
  threads: AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string, question?: string) => void;
};

/**
 * The Desk's front page.
 *
 * It is laid out like one: a dateline, a headline that says what the day
 * looks like, and then the question. The question is the point — the Desk is
 * a workspace you talk to, so the first thing on it is the thing you talk
 * into, not a directory of places to go. Everything under it is there to be
 * read at a glance and then left alone.
 *
 * The headline is composed from counts, never written by a model. A front
 * page that opens with a sentence nobody can trace is a front page people
 * stop reading, and the numbers here are cheap and exact.
 */
export function DeskHome({ agents, threads, isLoading, isStarting, onStart }: DeskHomeProps) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canManageAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { data: attention } = useAttentionSummary();
  // The briefing is written on a schedule, so before that hour there simply
  // is none. That is an ordinary answer, not a failure, and the page reads
  // the same without it.
  const briefingQuery = useQuery({ ...queries.briefing.today(), retry: false });
  const briefing = briefingQuery.data ?? null;
  const summaryQuery = usePendingDecisionSummary(canDecide);
  const waiting = summaryQuery.data?.total ?? attention?.agentDecisions ?? 0;
  const recent = threads.slice(0, RECENT_LIMIT);

  const dateline = useMemo(
    () =>
      new Intl.DateTimeFormat(undefined, {
        weekday: "long",
        month: "long",
        day: "numeric",
        timeZone: timezone,
      }).format(new Date(now * 1000)),
    [now, timezone],
  );

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-10 px-6 py-10">
      <header className="flex flex-col gap-1">
        <p className="text-muted-foreground text-xs">{dateline}</p>
        {/* The briefing's headline when there is one: it was written from
            figures gathered before a word of it, and every number in it was
            checked against them. The computed sentence below is what a
            morning reads like before the page has been written. */}
        <h1 className="text-2xl font-semibold tracking-tight text-balance">
          {briefing?.headline ||
            (canDecide && waiting > 0
              ? t(
                  "{0, plural, one {One decision is waiting on you.} other {# decisions are waiting on you.}}",
                  waiting,
                )
              : agents.length === 0
                ? t("Nothing is running here yet.")
                : t("Nothing is waiting on you."))}
        </h1>
        <p className="text-muted-foreground text-sm">
          {agents.length === 0
            ? t("The Desk comes alive once an agent is enabled.")
            : t("Ask an agent about the work in front of you. What it makes opens beside you.")}
        </p>
      </header>

      {agents.length > 0 && (
        <AskBox
          agents={agents}
          defaultAgentId={lastAgentId}
          disabled={isStarting}
          onAsk={(agentId, question) => onStart(agentId, question)}
        />
      )}

      {briefing && <BriefingPanel briefing={briefing} />}

      {canDecide && waiting > 0 && (
        <Link
          to="/desk/decisions"
          className={cn(
            "ui-focus-ring group border-desk-hairline flex items-center gap-3 rounded-surface border px-4 py-3",
            "hover:bg-surface-hover transition-colors",
          )}
        >
          <InboxIcon className="text-muted-foreground size-4 shrink-0" />
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
            <span className="text-sm">{t("Waiting on your decision")}</span>
            {summaryQuery.data?.byAgent.map((row) => (
              <Badge key={row.agentDefinitionId} variant="neutral" className="h-5 gap-1 px-1.5">
                <span className="max-w-40 truncate">{row.agentName || t("Retired agent")}</span>
                <span className="tabular-nums">{row.count}</span>
              </Badge>
            ))}
          </div>
          <ArrowRightIcon className="text-muted-foreground size-4 shrink-0 transition-transform group-hover:translate-x-0.5" />
        </Link>
      )}

      <section className="flex flex-col gap-3">
        <SectionLabel>{t("Who you can ask")}</SectionLabel>
        {isLoading ? (
          <div className="grid gap-2 sm:grid-cols-2">
            <Skeleton className="h-16" />
            <Skeleton className="h-16" />
          </div>
        ) : agents.length === 0 ? (
          <NoAgents canManageAgents={canManageAgents} />
        ) : (
          <div className="grid gap-2 sm:grid-cols-2">
            {agents.map((agent, index) => (
              <AgentCard
                key={agent.id}
                agent={agent}
                index={index}
                disabled={isStarting}
                onStart={() => onStart(agent.id)}
              />
            ))}
          </div>
        )}
      </section>

      {recent.length > 0 && (
        <section className="flex flex-col gap-3">
          <SectionLabel>{t("Where you left off")}</SectionLabel>
          <ul className="flex flex-col">
            {recent.map((thread) => {
              const touched = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;

              return (
                <li key={thread.id}>
                  <Link
                    to={`/desk/t/${thread.id}`}
                    className="hover:bg-surface-hover ui-focus-ring flex items-center gap-3 rounded-md px-2 py-1.5 transition-colors"
                  >
                    <span className="min-w-0 flex-1 truncate text-sm">
                      {thread.title || t("Untitled conversation")}
                    </span>
                    <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                      {formatSecondsAgo(now - touched)}
                    </span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </section>
      )}
    </div>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <h2 className="text-muted-foreground text-xs font-medium">{children}</h2>;
}

function AgentCard({
  agent,
  index,
  disabled,
  onStart,
}: {
  agent: AgentDefinitionRow;
  index: number;
  disabled: boolean;
  onStart: () => void;
}) {
  const t = useT();

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onStart}
      style={{ animationDelay: `${Math.min(index, 8) * 30}ms` }}
      className={cn(
        "animate-land ui-focus-ring ui-press border-desk-hairline group rounded-surface",
        "hover:bg-surface-hover flex items-center gap-3 border px-3 py-2.5 text-left",
        "transition-colors outline-none disabled:opacity-60",
      )}
    >
      <AgentTile agent={agent} size="lg" />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="flex items-center gap-1.5 text-sm font-medium">
          <span className="truncate">{agent.name}</span>
          <Badge variant="neutral" className="text-2xs h-4 px-1">
            {agent.toolNames.length === 0
              ? t("No task tools")
              : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
          </Badge>
        </span>
        {agent.description && (
          <span className="text-muted-foreground line-clamp-1 text-xs">{agent.description}</span>
        )}
      </span>
      <ArrowRightIcon className="text-muted-foreground size-4 shrink-0 transition-transform group-hover:translate-x-0.5" />
    </button>
  );
}

function NoAgents({ canManageAgents }: { canManageAgents: boolean }) {
  const t = useT();

  return (
    <div className="border-desk-hairline rounded-surface flex flex-col items-start gap-3 border p-4">
      <span className="bg-sunken text-muted-foreground flex size-9 items-center justify-center rounded-md">
        <BotIcon className="size-4" />
      </span>
      <div className="flex flex-col gap-1">
        <p className="text-sm font-medium">{t("No agents are available")}</p>
        <p className="text-muted-foreground text-xs">
          {canManageAgents
            ? t("Connect an AI provider and enable an agent in AI Control, then come back here.")
            : t("An administrator needs to connect an AI provider and enable an agent first.")}
        </p>
      </div>
      {canManageAgents && (
        <Button
          size="sm"
          variant="outline"
          nativeButton={false}
          render={<Link to="/admin/agent-control" />}
        >
          <PlugZapIcon className="size-3.5" />
          {t("Open AI control")}
        </Button>
      )}
    </div>
  );
}
