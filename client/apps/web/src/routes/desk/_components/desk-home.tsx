import { AgentTile } from "@/components/agent-identity/agent-tile";
import { AssistantMark } from "@/components/assistant/assistant-mark";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowRightIcon, BotIcon, InboxIcon, PlugZapIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useState } from "react";
import { Link } from "react-router";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const RECENT_LIMIT = 6;

export type DeskHomeProps = {
  agents: AgentDefinitionRow[];
  threads: AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string) => void;
};

/**
 * Where a day at the Desk starts: what is waiting on a decision, who can be
 * asked, and what was asked recently. It reads top to bottom like a morning
 * paper; the watchtower and the briefing take their place above the fold
 * when they land.
 */
export function DeskHome({ agents, threads, isLoading, isStarting, onStart }: DeskHomeProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [now] = useState(nowInSeconds);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canManageAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const summaryQuery = usePendingDecisionSummary(canDecide);
  const waiting = summaryQuery.data?.total ?? attention?.agentDecisions ?? 0;
  const recent = threads.slice(0, RECENT_LIMIT);

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-8">
      <m.header
        initial={reduceMotion ? false : { opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25 }}
        className="flex flex-col gap-2"
      >
        <AssistantMark className="text-foreground size-7" />
        <h1 className="text-lg font-semibold">{t("Your desk")}</h1>
        <p className="text-muted-foreground max-w-prose text-sm leading-relaxed">
          {t(
            "Ask any agent about the work in front of you, read what it produces beside the conversation, and decide what the agents have proposed.",
          )}
        </p>
      </m.header>

      {canDecide && (
        <section className="border-border flex flex-col gap-3 rounded-lg border p-4">
          <div className="flex items-center gap-3">
            <span className="bg-sunken flex size-9 shrink-0 items-center justify-center rounded-md">
              <InboxIcon className="size-4" />
            </span>
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="text-sm font-medium">
                {waiting === 0
                  ? t("Nothing is waiting on you")
                  : t("{0, plural, one {# decision waiting} other {# decisions waiting}}", waiting)}
              </span>
              <span className="text-muted-foreground text-xs">
                {waiting === 0
                  ? t("Changes an agent proposes will queue here for your approval.")
                  : t("Approve, change or reject what the agents proposed, one at a time or as a batch.")}
              </span>
            </div>
            <Button
              variant={waiting > 0 ? "default" : "outline"}
              size="sm"
              nativeButton={false}
              render={<Link to="/desk/decisions" />}
            >
              {t("Open decisions")}
              <ArrowRightIcon className="size-3.5" />
            </Button>
          </div>
          {summaryQuery.data && summaryQuery.data.byAgent.length > 0 && (
            <ul className="flex flex-wrap gap-1.5">
              {summaryQuery.data.byAgent.map((row) => (
                <li key={row.agentDefinitionId}>
                  <Badge variant="neutral" className="h-5 gap-1 px-1.5 text-xs">
                    <span className="truncate">{row.agentName || t("Retired agent")}</span>
                    <span className="tabular-nums">{row.count}</span>
                  </Badge>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      <section className="flex flex-col gap-2">
        <h2 className="text-muted-foreground text-xs font-medium">{t("Agents")}</h2>
        {isLoading ? (
          <div className="grid gap-2 sm:grid-cols-2">
            <Skeleton className="h-16" />
            <Skeleton className="h-16" />
          </div>
        ) : agents.length === 0 ? (
          <div className="border-border flex flex-col items-start gap-3 rounded-lg border p-4">
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
        ) : (
          <div className="grid gap-2 sm:grid-cols-2">
            {agents.map((agent, index) => (
              <m.button
                key={agent.id}
                type="button"
                disabled={isStarting}
                onClick={() => onStart(agent.id)}
                initial={reduceMotion ? false : { opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.2, delay: 0.03 * index }}
                className="ui-focus-ring ui-press border-border hover:bg-surface-hover group flex items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors outline-none disabled:opacity-60"
              >
                <AgentTile agent={agent} size="lg" />
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="flex items-center gap-1.5 text-sm font-medium">
                    <span className="truncate">{agent.name}</span>
                    <Badge variant="neutral" className="h-4 px-1 text-2xs">
                      {agent.toolNames.length === 0
                        ? t("Answers only")
                        : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
                    </Badge>
                  </span>
                  {agent.description && (
                    <span className="text-muted-foreground line-clamp-1 text-xs">
                      {agent.description}
                    </span>
                  )}
                </span>
                <ArrowRightIcon className="text-muted-foreground size-4 shrink-0 transition-transform group-hover:translate-x-0.5" />
              </m.button>
            ))}
          </div>
        )}
      </section>

      {recent.length > 0 && (
        <section className="flex flex-col gap-2">
          <h2 className="text-muted-foreground text-xs font-medium">
            {t("Pick up where you left off")}
          </h2>
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
                    <span className="text-muted-foreground shrink-0 text-xs">
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
