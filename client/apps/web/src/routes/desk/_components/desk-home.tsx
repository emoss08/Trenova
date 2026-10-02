import { AgentAsk, type AgentAskHandle } from "@/components/assistant/agent-ask";
import { useAskableAgent } from "@/components/assistant/use-askable-agent";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  partOfDay,
  resolveUserTimezone,
  toUserWallClock,
  type PartOfDay,
} from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { BotIcon, PlugZapIcon } from "lucide-react";
import { useMemo, useRef, useState, type CSSProperties } from "react";
import { Link } from "react-router";
import { DeskLedger, LEDGER_WATCHTOWER_ROWS } from "./desk-ledger";
import { DeskGreeting } from "./desk-greeting";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
/** The beat between one part of the page arriving and the next. */
const ENTRANCE_STEP_MS = 45;

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
  const threadsById = useMemo(
    () =>
      new Map(
        threads.map((thread) => [
          thread.id,
          { title: thread.title ?? "", agentDefinitionId: thread.agentDefinitionId },
        ]),
      ),
    [threads],
  );
  const liveQuery = useQuery({ ...queries.assistant.activeTurns(), retry: false });
  const live = liveQuery.data?.items ?? [];
  const watchtowerQuery = useQuery({
    ...queries.watchtower.feed({ unresolvedOnly: true, first: LEDGER_WATCHTOWER_ROWS }),
    enabled: canWatch,
    retry: false,
  });
  const watchtowerCounts = useQuery({ ...queries.watchtower.counts(), enabled: canWatch });

  const askable = useAskableAgent({ threads });
  const askRef = useRef<AgentAskHandle>(null);
  const heroRef = useRef<HTMLElement>(null);
  const noAgents = !isLoading && (agents.length === 0 || askable.noneAvailable);

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
          <DeskLedger
            now={now}
            agentsById={agentsById}
            threads={threadsById}
            canDecide={canDecide}
            canWatch={canWatch}
            waiting={waiting}
            summary={{ pending: summaryQuery.isPending, data: summaryQuery.data }}
            live={{ pending: liveQuery.isPending, items: live }}
            watchtower={{
              pending: watchtowerQuery.isPending,
              items: watchtowerQuery.data?.items ?? [],
              unseen: watchtowerCounts.data?.unseen ?? 0,
              critical: watchtowerCounts.data?.unseenCritical ?? 0,
            }}
            briefing={briefing}
            entrance={(step) => entrance(4 + step)}
            className="animate-rise"
            style={entrance(4)}
          />
        )}
      </div>
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
