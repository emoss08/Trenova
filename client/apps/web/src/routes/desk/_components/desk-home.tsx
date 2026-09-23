import { AgentAsk, type AgentAskHandle } from "@/components/assistant/agent-ask";
import { DeskMark } from "@/components/assistant/voice/desk-thinking";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { useAskableAgent } from "@/components/assistant/use-askable-agent";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentChoice, AgentDefinitionRow } from "@/lib/graphql/agent-definition";
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
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { BotIcon, PlugZapIcon } from "lucide-react";
import { useReducedMotion } from "motion/react";
import { useMemo, useRef, useState, type CSSProperties } from "react";
import { Link } from "react-router";
import { BriefingPanel } from "./briefing-panel";
import { DeskAgentDirectory } from "./desk-agent-directory";
import { DeskDecisionsCallout } from "./desk-decisions-callout";
import { DeskRecentConversations } from "./desk-recent-conversations";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const RECENT_LIMIT = 5;
/** The beat between one part of the page arriving and the next. */
const ENTRANCE_STEP_MS = 45;

export type DeskHomeProps = {
  agents: AgentDefinitionRow[];
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
 * The Desk's front page.
 *
 * It is laid out like one: a greeting and a dateline, a headline that says
 * what the day looks like, and then the question. The question is the point —
 * the Desk is a workspace you talk to, so the largest thing on it is the box
 * you talk into, with the chosen agent's own questions under it. What is
 * waiting on the person and where they left off sit beside it on a wide
 * screen and under it on a narrow one; the agents they could ask come last,
 * a page at a time.
 *
 * The headline is composed from counts, never written by a model, unless the
 * morning's briefing wrote one and checked every figure in it. A front page
 * that opens with a sentence nobody can trace is one people stop reading.
 *
 * The greeting sits beside a small drawing of a desk in the ink, set for the
 * part of the day: a sun low in the morning and high in the afternoon, and
 * in the evening a moon and the desk lamp switching on. It is the same desk
 * that works while an agent does, here at rest.
 *
 * Everything arrives once, in reading order, a beat apart — the desk, then
 * the greeting and the date, then the headline — and then holds still. The
 * chair settles in and the lamp catches as the page opens, and nothing here
 * moves again unless the person does something.
 */
export function DeskHome({ agents, threads, isLoading, isStarting, onStart }: DeskHomeProps) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const reduceMotion = useReducedMotion() ?? false;
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
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
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const askable = useAskableAgent({ threads });
  const askRef = useRef<AgentAskHandle>(null);
  const heroRef = useRef<HTMLDivElement>(null);
  const noAgents = !isLoading && (agents.length === 0 || askable.noneAvailable);

  const chooseAgent = (agent: AgentChoice) => {
    askable.choose(agent);
    heroRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
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
  // The same day, machine-readable, for the <time> that carries it.
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
        : t("Nothing is waiting on you."));

  const showDecisions = canDecide && waiting > 0;
  const showAside = showDecisions || recent.length > 0;

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col px-6 pt-12 pb-16 sm:px-10 lg:pt-16">
      <div
        className={cn(
          "grid grid-cols-1 gap-x-12 gap-y-10",
          showAside && "xl:grid-cols-[minmax(0,1fr)_19rem]",
        )}
      >
        <div ref={heroRef} className="flex min-w-0 scroll-mt-6 flex-col gap-7">
          <header className="flex flex-col gap-6">
            <div className="flex items-center gap-4">
              <span
                className="text-foreground-muted animate-rise flex shrink-0"
                style={entrance(0)}
              >
                <DeskMark
                  pose="idle"
                  size="xl"
                  timeOfDay={dayPart}
                  welcome
                  animate={!reduceMotion}
                />
              </span>
              <div className="flex min-w-0 flex-col gap-0.5">
                <p
                  className="text-foreground animate-rise truncate text-lg font-medium"
                  style={entrance(1)}
                >
                  {greeting(t, dayPart, firstName)}
                </p>
                <p className="text-foreground-subtle animate-rise text-sm" style={entrance(2)}>
                  <time dateTime={isoDate}>{dateline}</time>
                </p>
              </div>
            </div>
            <div className="flex flex-col gap-2">
              {/* The briefing's headline when there is one: it was written from
                  figures gathered before a word of it, and every number in it
                  was checked against them. The computed sentence is what a
                  morning reads like before the page has been written. */}
              <h1
                className="animate-rise max-w-2xl text-3xl font-semibold text-balance"
                style={entrance(3)}
              >
                {headline}
              </h1>
              <p
                className="text-foreground-muted animate-rise max-w-xl text-base text-pretty"
                style={entrance(4)}
              >
                {noAgents
                  ? t("The Desk comes alive once an agent is enabled.")
                  : t(
                      "Ask an agent about the work in front of you. What it makes opens beside you.",
                    )}
              </p>
            </div>
          </header>

          <div className="animate-rise" style={entrance(5)}>
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
                <div className="flex gap-1.5">
                  <Skeleton className="h-7 w-44 rounded-full" />
                  <Skeleton className="h-7 w-52 rounded-full" />
                  <Skeleton className="h-7 w-36 rounded-full" />
                </div>
              </div>
            )}
          </div>
        </div>

        {showAside && (
          <aside
            aria-label={t("Waiting on you and recent conversations")}
            className="animate-rise flex min-w-0 flex-col gap-8 xl:sticky xl:top-6 xl:col-start-2 xl:row-span-3 xl:row-start-1 xl:self-start"
            style={entrance(6)}
          >
            {showDecisions && (
              <DeskDecisionsCallout
                waiting={waiting}
                byAgent={summaryQuery.data?.byAgent ?? []}
                oldestAt={summaryQuery.data?.oldestAt ?? null}
                agentsById={agentsById}
                now={now}
              />
            )}
            <DeskRecentConversations
              threads={recent}
              agentsById={agentsById}
              liveThreadIds={liveThreadIds}
              now={now}
              className="-mx-2"
            />
          </aside>
        )}

        {briefing && (
          <div className="animate-rise min-w-0" style={entrance(7)}>
            <BriefingPanel briefing={briefing} />
          </div>
        )}

        {!noAgents && (
          <div className="animate-rise min-w-0" style={entrance(8)}>
            <DeskAgentDirectory
              recency={askable.recency}
              selectedId={askable.agent?.id ?? null}
              disabled={isStarting}
              onChoose={chooseAgent}
            />
          </div>
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
    <div className="border-desk-hairline rounded-surface flex flex-col items-start gap-3 border border-dashed p-5">
      <span className="bg-sunken text-muted-foreground flex size-9 items-center justify-center rounded-md">
        <BotIcon className="size-4" />
      </span>
      <div className="flex flex-col gap-1">
        <p className="text-sm font-semibold">{t("No agents are available")}</p>
        <p className="text-muted-foreground text-sm">
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
