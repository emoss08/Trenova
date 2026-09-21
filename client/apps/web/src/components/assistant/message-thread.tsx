import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantPageContext, AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { ArrowRightIcon, XIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AssistantAgentProvider } from "@/components/agent-identity/agent-context";
import { Composer } from "./composer";
import {
  AgentAvatar,
  AssistantEntry,
  DayDivider,
  DeclinedTurn,
  PageContextChip,
  RefusalNotice,
  UserTurn,
} from "./message-items";
import { PlanCard } from "./plan-card";
import { groupPlans } from "./plan-state";
import { ProposalCard } from "./proposal-card";
import { groupProposalsByMessage } from "./proposal-state";
import { StreamingTurn } from "./streaming-turn";
import { suggestionsFor, type Suggestion } from "./suggestions";
import { arrivedSince, highestSequence, withDayMarkers } from "./thread-rows";
import { groupThread } from "./thread-view";
import { useAssistantTurn } from "./use-assistant-turn";
import { usePageContext } from "./use-page-context";
import { useThreadHistory } from "./use-thread-history";
import { VirtualThread, type VirtualThreadRow } from "./virtual-thread";

/**
 * Space between the last message and the composer's fade, beyond the
 * composer's own height: the fade is a band, and a line of text inside it
 * reads as cut off.
 */
const COMPOSER_CLEARANCE = 16;

export function MessageThread({
  thread,
  agent,
  expanded,
  onPickAgent,
  onStartNew,
}: {
  thread: AssistantThread;
  agent: AgentDefinitionRow | null;
  expanded: boolean;
  onPickAgent?: () => void;
  /** Starts a fresh conversation with the same agent; offered when this one is full. */
  onStartNew?: () => void;
}) {
  const t = useT();
  const dismissed = useAssistantStore((state) => state.dismissedSuggestions);
  const dismissSuggestion = useAssistantStore((state) => state.dismissSuggestion);
  const draft = useAssistantStore((state) => state.drafts[thread.id] ?? "");
  const setDraft = useAssistantStore((state) => state.setDraft);
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const onDraftChange = useCallback(
    (value: string) => setDraft(thread.id, value),
    [setDraft, thread.id],
  );

  const history = useThreadHistory(thread.id);
  const { messages } = history;

  // What the thread held when it was opened. Only a message numbered past it
  // is an arrival to this reader, and only arrivals rise into place: a page
  // of older history or a row scrolling back into the window does not.
  const openedAt = useRef<number | null>(null);
  if (openedAt.current === null && !history.isLoading) {
    openedAt.current = highestSequence(messages);
  }
  const arrivals = useMemo(
    () => arrivedSince(openedAt.current ?? Number.POSITIVE_INFINITY, messages),
    [messages],
  );
  // Read once per mount: day markers are relative to when the thread was
  // opened, and a clock read during render would make every render impure.
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  // Proposals are fetched rather than taken from the send response: they outlive
  // the turn that raised them, so reopening a thread has to show what is still
  // waiting on a decision.
  const providersQuery = useQuery(queries.assistant.providers());
  const providers = useMemo(() => providersQuery.data ?? [], [providersQuery.data]);

  // The thread carries the choice, so a reload reopens on the same model, and a
  // local copy makes picking one immediate rather than waiting for the turn
  // that saves it. The pick is tagged with the thread it was made in: opening
  // another conversation falls back to that thread's own stored choice without
  // an effect resetting it a render later.
  // null means nothing picked in this session yet — not "picked Auto". Auto is
  // an empty provider id and a real choice, so the two have to stay distinct or
  // clearing a model would silently restore the stored one.
  const [picked, setPicked] = useState<{ threadId: string; providerId: string } | null>(null);
  const providerId =
    picked && picked.threadId === thread.id
      ? picked.providerId
      : (thread.preferredProviderId ?? "");
  const setProviderId = useCallback(
    (next: string) => setPicked({ threadId: thread.id, providerId: next }),
    [thread.id],
  );

  const proposalsQuery = useQuery(queries.assistant.proposals(thread.id));
  const plansQuery = useQuery(queries.assistant.plans(thread.id));
  // A plan's steps are shown inside the plan and nowhere else; only the
  // proposals outside any plan are grouped under their turns on their own.
  const {
    byMessage: plansByMessage,
    orphans: loosePlans,
    standalone,
  } = groupPlans(plansQuery.data?.results ?? [], proposalsQuery.data?.results ?? [], messages);
  const { byMessage: proposalsByMessage, orphans: looseProposals } = groupProposalsByMessage(
    standalone,
    messages,
  );

  const entries = useMemo(() => groupThread(messages), [messages]);

  // A question the assistant asked is settled by whatever the person said next,
  // whether they clicked one of its options or typed something else entirely.
  // Comparing positions rather than tracking which button was pressed is what
  // makes a reopened thread render the same as a live one.
  const latestUserSequence = useMemo(
    () =>
      messages.reduce(
        (latest, message) =>
          message.role === "User" ? Math.max(latest, message.sequence) : latest,
        -1,
      ),
    [messages],
  );

  const getPageContext = usePageContext();
  const [contextIncluded, setContextIncluded] = useState(true);
  const pageContext = getPageContext();
  // A person who drops the page chip means it: the turn is sent without it
  // rather than with a note saying they would rather it were not there.
  const getTurnContext = useCallback(
    () => (contextIncluded ? getPageContext() : null),
    [contextIncluded, getPageContext],
  );
  const { turn, isActive, send, stop, dismiss, retry } = useAssistantTurn(
    thread.id,
    getTurnContext,
  );

  // An answer to the assistant's question is an ordinary message. Sending it
  // that way is what keeps a clicked answer and a typed one the same thing:
  // nothing new is stored, and the thread reads identically either way.
  const answer = useCallback(
    (value: string) => void send(value, undefined, providerId),
    [send, providerId],
  );

  // The composer floats over the bottom of the thread, so the last message has
  // to be padded clear of it and the jump-to-latest button lifted above it. The
  // textarea grows to eight rows, which is why this is measured, not a constant.
  const composerRef = useRef<HTMLDivElement>(null);
  const [composerHeight, setComposerHeight] = useState(0);

  useEffect(() => {
    const element = composerRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }

    const observer = new ResizeObserver(([entry]) => {
      setComposerHeight(entry.target.getBoundingClientRect().height);
    });
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  const agentUnavailable = agent === null;
  const threadFull = history.length.state === "full";
  const isEmpty = !history.isLoading && entries.length === 0 && turn === null;
  // The starter questions are listed on an empty thread and behind a slash
  // in the composer at any time; a dismissed one stays dismissed in both.
  const suggestions = useMemo(
    () =>
      agent
        ? suggestionsFor(agent.template).filter((item) => !dismissed.includes(item.prompt))
        : [],
    [agent, dismissed],
  );

  // Every row is a closure over its entry, keyed by the message it shows, so
  // the window can measure and place it without knowing what it is.
  const rows = useMemo<VirtualThreadRow[]>(() => {
    const list: VirtualThreadRow[] = withDayMarkers(entries, now, timezone).map((item) => {
      if (item.kind === "day") {
        return {
          key: item.key,
          render: () => <DayDivider at={item.at} daysAgo={item.daysAgo} />,
        };
      }
      const { entry } = item;
      const arrived = arrivals.has(entry.message.id);
      return {
        key: entry.message.id,
        render: () => (
          <div className={cn(arrived && "animate-rise")}>
            {entry.kind === "user" ? (
              <UserTurn
                content={entry.message.content}
                sentAt={entry.message.createdAt}
                pageContext={entry.message.pageContext}
                onResend={
                  entry.message.sequence === latestUserSequence
                    ? () => void send(entry.message.content, undefined, providerId)
                    : undefined
                }
              />
            ) : entry.kind === "declined" ? (
              <DeclinedTurn content={entry.message.content} sentAt={entry.message.createdAt} />
            ) : entry.kind === "refusal" ? (
              <RefusalNotice message={entry.message.content} />
            ) : (
              <AssistantEntry
                entry={entry}
                proposals={proposalsByMessage.get(entry.message.id) ?? []}
                plans={plansByMessage.get(entry.message.id) ?? []}
                threadId={thread.id}
                latestUserSequence={latestUserSequence}
                onAnswer={answer}
              />
            )}
          </div>
        ),
      };
    });

    // A proposal whose turn is no longer in the visible thread is shown here
    // rather than dropped: a pending change nobody can see is worse than one
    // shown out of position.
    for (const group of loosePlans) {
      list.push({
        key: `plan-${group.plan.id}`,
        render: () => <PlanCard plan={group.plan} steps={group.steps} threadId={thread.id} />,
      });
    }
    for (const proposal of looseProposals) {
      list.push({
        key: `proposal-${proposal.id}`,
        render: () => <ProposalCard proposal={proposal} threadId={thread.id} />,
      });
    }

    if (turn) {
      list.push({
        key: "turn-in-progress",
        render: () => (
          <div className="animate-rise">
            <StreamingTurn turn={turn} onRetry={retry} onDismiss={dismiss} onAnswer={answer} />
          </div>
        ),
      });
    }

    return list;
  }, [
    answer,
    arrivals,
    dismiss,
    entries,
    latestUserSequence,
    loosePlans,
    looseProposals,
    now,
    plansByMessage,
    proposalsByMessage,
    providerId,
    retry,
    send,
    thread.id,
    timezone,
    turn,
  ]);

  return (
    <AssistantAgentProvider agent={agent}>
      <div className="relative flex min-h-0 flex-1 flex-col">
        {history.isLoading ? (
          <div className={cn("flex flex-1 flex-col gap-4", expanded ? "px-4 py-5" : "px-3 py-4")}>
            <Skeleton className="ml-auto h-10 w-2/5" />
            <Skeleton className="h-20 w-3/5" />
            <Skeleton className="ml-auto h-10 w-1/3" />
          </div>
        ) : isEmpty ? (
          <div className="flex min-h-0 flex-1 flex-col" style={{ paddingBottom: composerHeight }}>
            <EmptyThread
              agent={agent}
              suggestions={suggestions}
              pageContext={contextIncluded ? pageContext : null}
              onPick={(prompt) => void send(prompt, undefined, providerId)}
              onDismiss={dismissSuggestion}
            />
          </div>
        ) : (
          <VirtualThread
            rows={rows}
            hasOlder={history.hasOlder}
            isLoadingOlder={history.isLoadingOlder}
            onLoadOlder={history.loadOlder}
            paddingBottom={composerHeight + COMPOSER_CLEARANCE}
            className={expanded ? "px-4" : "px-3"}
            contentClassName={expanded ? "max-w-3xl pt-5" : "pt-4"}
            rowClassName={expanded ? "pb-5" : "pb-4"}
          />
        )}

        <Composer
          ref={composerRef}
          onSend={(content) => void send(content, undefined, providerId)}
          onStop={stop}
          active={isActive}
          disabled={agentUnavailable || threadFull}
          disabledReason={
            threadFull
              ? t("This conversation is full. Start a new one to continue.")
              : t("This agent has been disabled, so the conversation cannot continue.")
          }
          notice={
            history.length.state !== "open" ? (
              <ThreadLengthNotice
                state={history.length.state}
                total={history.total}
                limit={history.limit}
                onStartNew={onStartNew}
              />
            ) : null
          }
          placeholder={
            agent
              ? t("Message {0}…", agent.name)
              : t("Ask about a shipment, a driver, or how to do something…")
          }
          agent={agent}
          onPickAgent={onPickAgent}
          pageContext={pageContext}
          contextIncluded={contextIncluded}
          onToggleContext={() => setContextIncluded((value) => !value)}
          providers={providers}
          providerId={providerId}
          onPickProvider={setProviderId}
          suggestions={suggestions}
          draft={draft}
          onDraftChange={onDraftChange}
          compact={!expanded}
        />
      </div>
    </AssistantAgentProvider>
  );
}

/**
 * The first thing a reader sees in a new conversation: what this agent is
 * for, what it can see, and a few questions it is good at. The questions are
 * a list, not chips: each is a sentence a person can read and choose, and
 * closing one is a decision that is remembered.
 */
function EmptyThread({
  agent,
  suggestions,
  pageContext,
  onPick,
  onDismiss,
}: {
  agent: AgentDefinitionRow | null;
  suggestions: readonly Suggestion[];
  pageContext: AssistantPageContext | null;
  onPick: (prompt: string) => void;
  onDismiss: (prompt: string) => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-end gap-5 px-4 py-6">
      <m.div
        initial={reduceMotion ? false : { opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2 }}
        className="flex flex-col gap-2"
      >
        <AgentAvatar size="lg" />
        <p className="text-base font-semibold">
          {agent ? t("What do you need from {0}?", agent.name) : t("Start a conversation")}
        </p>
        <p className="text-muted-foreground text-sm leading-relaxed">
          {agent?.description ||
            t(
              "Ask about a shipment, a driver, or how to do something in Trenova. The assistant can look records up and propose changes for you to approve.",
            )}
        </p>
        {pageContext && (pageContext.title !== "" || pageContext.entityType !== "") && (
          <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
            {t("Can see")}
            <PageContextChip context={pageContext} />
          </p>
        )}
      </m.div>

      {suggestions.length > 0 && (
        <ul className="flex flex-col gap-1">
          {suggestions.map((suggestion, index) => (
            <m.li
              key={suggestion.prompt}
              initial={reduceMotion ? false : { opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.18, delay: 0.05 + index * 0.04 }}
              className="group/suggestion flex items-center gap-1"
            >
              <button
                type="button"
                onClick={() => onPick(suggestion.prompt)}
                className="hover:bg-surface-hover ui-focus-ring flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors"
              >
                <ArrowRightIcon className="text-muted-foreground size-3.5 shrink-0 transition-transform group-hover/suggestion:translate-x-0.5" />
                <span className="min-w-0 flex-1 truncate">{t(suggestion.label)}</span>
              </button>
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label={t("Dismiss suggestion")}
                className="text-muted-foreground hover:text-foreground opacity-0 transition-opacity group-hover/suggestion:opacity-100 focus-visible:opacity-100"
                onClick={() => onDismiss(suggestion.prompt)}
              >
                <XIcon className="size-3" />
              </Button>
            </m.li>
          ))}
        </ul>
      )}
    </div>
  );
}

/**
 * Where a conversation stands against its length, said before the wall.
 *
 * The server refuses a turn once the thread holds its limit; a person who
 * only learns that from the refusal has already typed the question. A long
 * thread is also a slow one to open, which is the reason for the limit.
 */
function ThreadLengthNotice({
  state,
  total,
  limit,
  onStartNew,
}: {
  state: "long" | "full";
  total: number;
  limit: number;
  onStartNew?: () => void;
}) {
  const t = useT();

  return (
    <div className="text-muted-foreground flex items-center justify-between gap-3 px-1 text-xs">
      <span>
        {state === "full"
          ? t("This conversation has reached {0} messages and is now read-only.", limit)
          : t("Long conversation: {0} of {1} messages. A new one will open faster.", total, limit)}
      </span>
      {onStartNew && (
        <Button
          variant={state === "full" ? "default" : "ghost"}
          size="xs"
          className="shrink-0"
          onClick={onStartNew}
        >
          {t("Start a new conversation")}
        </Button>
      )}
    </div>
  );
}
