import { AgentAsk, type AgentAskHandle } from "@/components/assistant/agent-ask";
import { useAskableAgent } from "@/components/assistant/use-askable-agent";
import { WorkingDot } from "@/components/assistant/voice/working-dot";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { queries } from "@/lib/queries";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  formatShortAge,
  partOfDay,
  resolveUserTimezone,
  toUserWallClock,
  type PartOfDay,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRightIcon,
  BotIcon,
  InboxIcon,
  PlugZapIcon,
  RadarIcon,
  type LucideIcon,
} from "lucide-react";
import { useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { Link } from "react-router";
import { BriefingPanel } from "./briefing-panel";
import { DeskGreeting } from "./desk-greeting";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
/** The beat between one part of the page arriving and the next. */
const ENTRANCE_STEP_MS = 45;
/** Agents shown on the shelf before the rest are counted. */
const SHELF_AGENTS = 6;
/** Watchtower items the card shows before the rest are counted. */
const WATCHTOWER_ROWS = 4;
/** Agents named under the waiting count before the rest are counted. */
const NAMED_AGENTS = 3;

export type DeskHomeProps = {
  agents: AgentChoice[];
  threads: AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string, question?: string) => void;
};

function greeting(t: TranslateFn, dayPart: PartOfDay, firstName: string): string {
  switch (dayPart) {
    case "morning":
      return firstName ? t("Good morning, {0}", firstName) : t("Good morning");
    case "afternoon":
      return firstName ? t("Good afternoon, {0}", firstName) : t("Good afternoon");
    default:
      return firstName ? t("Good evening, {0}", firstName) : t("Good evening");
  }
}

/** Staggers a part of the page in behind the one above it. */
function entrance(step: number): CSSProperties {
  return { animationDelay: `${step * ENTRANCE_STEP_MS}ms` };
}

/**
 * The Desk's front page: the question first, then the day around it.
 *
 * It opens on the box you talk into, under a greeting that says what the
 * day looks like in one line, and under that the desk itself: what is
 * waiting on you, what is being answered right now, what the watchtower
 * has seen, the morning's page, and the agents you could ask. Every figure
 * is one the rest of the product already shows, and every card opens onto
 * the page that holds its rows. Nothing here is a model's sentence unless
 * the briefing wrote it and checked it.
 *
 * Everything arrives once, in reading order, a beat apart, and holds still.
 */
export function DeskHome({ agents, threads, isLoading, isStarting, onStart }: DeskHomeProps) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const { allowed: canDecide, isLoading: permissionsLoading } = usePermission(
    Resource.AgentProposal,
    Operation.Read,
  );
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: canManageAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const briefingQuery = useQuery({ ...queries.briefing.today(), retry: false });
  const briefing = briefingQuery.data ?? null;
  const summaryQuery = usePendingDecisionSummary(canDecide);
  const waiting = summaryQuery.data?.total ?? attention?.agentDecisions ?? 0;
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);
  const liveQuery = useQuery({ ...queries.assistant.activeTurns(), retry: false });
  const live = liveQuery.data?.items ?? [];
  const watchtowerQuery = useQuery({
    ...queries.watchtower.feed({ unresolvedOnly: true, first: WATCHTOWER_ROWS }),
    enabled: canWatch,
    retry: false,
  });
  const watchtowerCounts = useQuery({ ...queries.watchtower.counts(), enabled: canWatch });

  const askable = useAskableAgent({ threads });
  const askRef = useRef<AgentAskHandle>(null);
  const heroRef = useRef<HTMLElement>(null);
  const noAgents = !isLoading && (agents.length === 0 || askable.noneAvailable);

  const chooseAgent = (agent: AgentChoice) => {
    askable.choose(agent);
    heroRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    askRef.current?.focus();
  };

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
  const isoDate = useMemo(
    () =>
      new Intl.DateTimeFormat("en-CA", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        timeZone: timezone,
      }).format(new Date(now * 1000)),
    [now, timezone],
  );
  const hour = toUserWallClock(now, timezone)?.getHours() ?? 9;
  const dayPart = partOfDay(hour);
  const firstName = user?.name?.trim().split(/\s+/)[0] ?? "";

  const headline =
    briefing?.headline ||
    (canDecide && waiting > 0
      ? t(
          "{0, plural, one {One decision is waiting on you.} other {# decisions are waiting on you.}}",
          waiting,
        )
      : noAgents
        ? t("Nothing is running here yet.")
        : t("How can I help today?"));

  const headlinePending =
    isLoading ||
    permissionsLoading ||
    briefingQuery.isPending ||
    (canDecide && summaryQuery.isPending);

  const shelf = agents.slice(0, SHELF_AGENTS);

  return (
    <div className="w-full overflow-x-clip">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-6 pt-10 pb-16 sm:px-10">
        <section
          ref={heroRef}
          aria-label={t("Ask an agent")}
          className="flex scroll-mt-6 flex-col items-center gap-7"
        >
          <DeskGreeting
            greeting={greeting(t, dayPart, firstName)}
            dateline={dateline}
            isoDate={isoDate}
            entrance={entrance}
            headline={
              headlinePending ? (
                <Skeleton
                  data-part="headline-skeleton"
                  className="mx-auto h-7 w-3/4 max-w-md align-middle"
                />
              ) : (
                headline
              )
            }
          />

          <div className="animate-rise w-full max-w-2xl" style={entrance(3)}>
            {noAgents ? (
              <NoAgents canManageAgents={canManageAgents} />
            ) : askable.agent ? (
              <AgentAsk
                ref={askRef}
                agent={askable.agent}
                onAgentChange={askable.choose}
                recentIds={askable.recency.ids}
                lastUsedAt={askable.recency.lastUsedAt}
                disabled={isStarting}
                onAsk={(agentId, question) => onStart(agentId, question)}
              />
            ) : askable.choices.isError ? (
              <AgentsUnavailable onRetry={askable.choices.refetch} />
            ) : (
              <div className="flex flex-col gap-3" aria-busy>
                <Skeleton className="rounded-surface h-26" />
                <div className="flex justify-center gap-1.5">
                  <Skeleton className="h-7 w-44 rounded-full" />
                  <Skeleton className="h-7 w-52 rounded-full" />
                  <Skeleton className="h-7 w-36 rounded-full" />
                </div>
              </div>
            )}
          </div>
        </section>

        {!noAgents && (
          <section
            aria-label={t("Waiting on you, your day and your agents")}
            className="grid grid-cols-1 items-start gap-4 md:grid-cols-2 xl:grid-cols-12"
          >
            {canDecide && (
              <DeskCard
                step={4}
                icon={InboxIcon}
                title={t("Waiting on you")}
                figure={waiting}
                to="/desk/decisions"
                action={t("Review decisions")}
                className="xl:col-span-4"
              >
                {summaryQuery.isPending ? (
                  <CardSkeleton rows={3} />
                ) : waiting === 0 ? (
                  <CardEmpty>
                    {t("Nothing is waiting on you. Every proposal has been decided.")}
                  </CardEmpty>
                ) : (
                  <ul className="flex flex-col gap-1.5">
                    {(summaryQuery.data?.byAgent ?? []).slice(0, NAMED_AGENTS).map((row) => (
                      <li key={row.agentDefinitionId} className="flex items-center gap-2 text-sm">
                        <AgentTile
                          agent={
                            agentsById.get(row.agentDefinitionId) ?? {
                              id: row.agentDefinitionId,
                              name: row.agentName,
                            }
                          }
                          size="xs"
                        />
                        <span className="min-w-0 flex-1 truncate">
                          {row.agentName || t("Retired agent")}
                        </span>
                        <span className="text-foreground-subtle tabular-nums">{row.count}</span>
                      </li>
                    ))}
                    {(summaryQuery.data?.byAgent.length ?? 0) > NAMED_AGENTS && (
                      <li className="text-foreground-subtle pl-6 text-xs">
                        {t(
                          "{0, plural, one {and one more agent} other {and # more agents}}",
                          (summaryQuery.data?.byAgent.length ?? 0) - NAMED_AGENTS,
                        )}
                      </li>
                    )}
                    {summaryQuery.data?.oldestAt ? (
                      <li className="text-foreground-subtle pt-1 text-xs tabular-nums">
                        {t("Oldest {0}", formatShortAge(now - summaryQuery.data.oldestAt))}
                      </li>
                    ) : null}
                  </ul>
                )}
              </DeskCard>
            )}

            <DeskCard
              step={5}
              icon={BotIcon}
              title={t("Replying now")}
              figure={live.length}
              className="xl:col-span-4"
            >
              {liveQuery.isPending ? (
                <CardSkeleton rows={2} />
              ) : live.length === 0 ? (
                <CardEmpty>{t("No agent is writing a reply right now.")}</CardEmpty>
              ) : (
                <ul className="flex flex-col gap-1">
                  {live.map((turn) => {
                    const thread = threads.find((item) => item.id === turn.threadId);
                    const agent = thread ? agentsById.get(thread.agentDefinitionId) : null;

                    return (
                      <li key={turn.turnId}>
                        <Link
                          to={conversationPath(turn.threadId)}
                          className="ui-focus-ring hover:bg-surface-hover -mx-2 flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors"
                        >
                          <WorkingDot working still />
                          <span className="min-w-0 flex-1 truncate">
                            {turn.threadTitle || thread?.title || t("Untitled conversation")}
                          </span>
                          <span className="text-foreground-subtle flex shrink-0 items-center gap-1.5 text-xs">
                            <AgentTile agent={agent ?? null} size="xs" className="size-3.5" />
                            {formatShortAge(now - turn.startedAt)}
                          </span>
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              )}
            </DeskCard>

            {canWatch && (
              <DeskCard
                step={6}
                icon={RadarIcon}
                title={t("Watchtower")}
                figure={watchtowerCounts.data?.unseen ?? 0}
                figureTone={(watchtowerCounts.data?.unseenCritical ?? 0) > 0 ? "danger" : undefined}
                to="/desk/watchtower"
                action={t("Open the watchtower")}
                className="xl:col-span-4"
              >
                {watchtowerQuery.isPending ? (
                  <CardSkeleton rows={3} />
                ) : (watchtowerQuery.data?.items.length ?? 0) === 0 ? (
                  <CardEmpty>{t("Nothing unresolved. The watchtower is quiet.")}</CardEmpty>
                ) : (
                  <ul className="flex flex-col gap-1">
                    {watchtowerQuery.data?.items.slice(0, WATCHTOWER_ROWS).map((item) => (
                      <WatchtowerRow key={item.id} item={item} now={now} />
                    ))}
                  </ul>
                )}
              </DeskCard>
            )}

            {briefing && (
              <div className="animate-rise xl:col-span-7" style={entrance(7)}>
                <BriefingPanel briefing={briefing} />
              </div>
            )}

            <DeskCard
              step={8}
              icon={BotIcon}
              title={t("Agents")}
              figure={agents.length}
              className={cn(briefing ? "xl:col-span-5" : "xl:col-span-12")}
            >
              {isLoading ? (
                <CardSkeleton rows={4} />
              ) : (
                <ul
                  className={cn(
                    "flex flex-col gap-0.5",
                    !briefing && "md:grid md:grid-cols-2 xl:grid-cols-3",
                  )}
                >
                  {shelf.map((agent) => (
                    <li key={agent.id}>
                      <button
                        type="button"
                        disabled={isStarting}
                        onClick={() => chooseAgent(agent)}
                        className="group/agent ui-focus-ring hover:bg-surface-hover -mx-2 flex w-[calc(100%+1rem)] items-center gap-3 rounded-md px-2 py-2 text-left transition-colors disabled:opacity-60"
                      >
                        <AgentTile agent={agent} size="sm" />
                        <span className="grid min-w-0 flex-1 leading-tight">
                          <span className="truncate text-sm font-medium">{agent.name}</span>
                          <span className="text-foreground-subtle truncate text-xs">
                            {agent.description}
                          </span>
                        </span>
                        <ArrowRightIcon className="text-foreground-subtle size-3.5 shrink-0 opacity-0 transition-[opacity,translate] group-hover/agent:translate-x-0.5 group-hover/agent:opacity-100" />
                      </button>
                    </li>
                  ))}
                  {agents.length > SHELF_AGENTS && (
                    <li className="text-foreground-subtle px-0 pt-1 text-xs">
                      {t(
                        "{0, plural, one {and one more agent} other {and # more agents}}",
                        agents.length - SHELF_AGENTS,
                      )}
                    </li>
                  )}
                </ul>
              )}
            </DeskCard>
          </section>
        )}
      </div>
    </div>
  );
}

/**
 * One card of the desk: a quiet frame with a mark, a title, the one figure
 * that matters, and the way to the page behind it.
 */
function DeskCard({
  step,
  icon: Icon,
  title,
  figure,
  figureTone,
  to,
  action,
  className,
  children,
}: {
  step: number;
  icon: LucideIcon;
  title: string;
  figure?: number;
  figureTone?: "danger";
  to?: string;
  action?: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section
      aria-label={title}
      style={entrance(step)}
      className={cn(
        "animate-rise bg-card ring-foreground/10 flex min-h-40 flex-col gap-3 rounded-lg p-4 ring-1",
        className,
      )}
    >
      <header className="flex items-center gap-2">
        <Icon className="text-foreground-subtle size-4 shrink-0" />
        <h2 className="text-foreground-muted min-w-0 flex-1 truncate text-sm font-medium">
          {title}
        </h2>
        {figure !== undefined && (
          <span
            className={cn(
              "text-lg font-semibold tabular-nums",
              figureTone === "danger" ? "text-danger" : "text-foreground",
            )}
          >
            {figure > 999 ? "999+" : figure}
          </span>
        )}
      </header>
      <div className="flex min-h-0 flex-1 flex-col">{children}</div>
      {to && action && (
        <Button
          nativeButton={false}
          variant="ghost"
          size="sm"
          render={<Link to={to} />}
          className="group/card text-foreground-muted hover:text-foreground -mx-2 -mb-1.5 justify-start px-2"
        >
          {action}
          <ArrowRightIcon className="size-3.5 transition-transform group-hover/card:translate-x-0.5" />
        </Button>
      )}
    </section>
  );
}

function WatchtowerRow({ item, now }: { item: WatchtowerItem; now: number }) {
  return (
    <li>
      <Link
        to={item.path || "/desk/watchtower"}
        className="ui-focus-ring hover:bg-surface-hover -mx-2 flex items-start gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors"
      >
        <span
          aria-hidden
          className={cn(
            "mt-1.5 size-1.5 shrink-0 rounded-full",
            item.severity === "Critical"
              ? "bg-danger"
              : item.severity === "Warning"
                ? "bg-warning"
                : "bg-foreground-subtle",
          )}
        />
        <span className="grid min-w-0 flex-1 leading-tight">
          <span className="truncate">{item.title}</span>
          <span className="text-foreground-subtle truncate text-xs">
            {item.kindLabel} · {formatShortAge(now - item.occurredAt)}
          </span>
        </span>
      </Link>
    </li>
  );
}

function CardEmpty({ children }: { children: ReactNode }) {
  return (
    <p className="text-foreground-subtle flex flex-1 items-center text-sm text-pretty">
      {children}
    </p>
  );
}

function CardSkeleton({ rows }: { rows: number }) {
  return (
    <div className="flex flex-col gap-2" aria-busy>
      {Array.from({ length: rows }, (_, index) => (
        <Skeleton key={index} className="h-5" style={{ width: `${90 - index * 15}%` }} />
      ))}
    </div>
  );
}

function AgentsUnavailable({ onRetry }: { onRetry: () => void }) {
  const t = useT();

  return (
    <div className="border-desk-hairline rounded-surface flex items-center gap-3 border px-4 py-3">
      <p className="text-muted-foreground min-w-0 flex-1 text-sm">
        {t("The agents could not be loaded.")}
      </p>
      <Button size="sm" variant="outline" onClick={onRetry}>
        {t("Try again")}
      </Button>
    </div>
  );
}

function NoAgents({ canManageAgents }: { canManageAgents: boolean }) {
  const t = useT();

  return (
    <div className="border-desk-hairline rounded-surface flex flex-col items-center gap-3 border border-dashed p-6 text-center">
      <span className="bg-sunken text-muted-foreground flex size-9 items-center justify-center rounded-md">
        <BotIcon className="size-4" />
      </span>
      <div className="flex flex-col gap-1">
        <p className="text-sm font-semibold">{t("No agents are available")}</p>
        <p className="text-muted-foreground max-w-md text-sm text-pretty">
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
