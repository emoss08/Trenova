import { AssistantAgentProvider } from "@/components/agent-identity/agent-context";
import { useCalendarNow } from "@/hooks/use-calendar-now";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type {
  AssistantArtifact,
  AssistantArtifactEvent,
  AssistantPageContext,
  AssistantThread,
} from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { EASE_SETTLE } from "@/lib/motion";
import { InfoCircleIcon } from "@trenova/shared/components/icons";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { AgentStarters } from "./agent-starters";
import { ApprovalDock } from "./approval-dock";
import { ArtifactOpenerProvider } from "./artifact-opener";
import { Composer } from "./composer";
import { DeferredDecisionsPill } from "./deferred-decisions-pill";
import { PlanRecord, ProposalRecord } from "./decision-record";
import { DecisionFollowUpProvider } from "./decision-follow-up";
import {
  AgentAvatar,
  AssistantEntry,
  DayDivider,
  CompactionNote,
  DecisionNote,
  DeclinedTurn,
  PageContextChip,
  RefusalNotice,
  UserTurn,
} from "./message-items";
import { type ModelSwitchNotice as ModelSwitchNoticeValue } from "./model-switch";
import { ReadOnlyThreadNotice } from "./read-only-thread-notice";
import { StreamingTurn } from "./streaming-turn";
import { type Suggestion } from "./suggestions";
import { arrivedSince, highestSequence, withDayMarkers } from "./thread-rows";
import { useThreadModel, type PageBinding, type PageRequest } from "./use-thread-model";
import { VirtualThread, type VirtualThreadRow } from "./virtual-thread";

export type { PageBinding, PageRequest } from "./use-thread-model";
import { AgentGutter, agentSpineColor } from "./voice/agent-gutter";

/**
 * Space between the last message and the composer's fade, beyond the
 * composer's own height: the fade is a band, and a line of text inside it
 * reads as cut off.
 */
const COMPOSER_CLEARANCE = 16;

/**
 * The height of the fade band at the top of the composer (h-10 and h-6 in
 * composer.tsx). The jump-to-latest control sits in that band, where the
 * thread is already fading under the box, rather than over the lines above
 * it that the reader is trying to read.
 */
const COMPOSER_FADE = 40;
const COMPOSER_FADE_COMPACT = 24;

/**
 * How many of the rows a thread opens on rise one after another, from the
 * top of the window down, and how far apart. The list is anchored to its
 * end, so the last few rows are the ones in view; anything above them is
 * off screen and rises at once if it is ever drawn.
 */
const OPENING_STAGGER_ROWS = 6;
const OPENING_STAGGER_MS = 30;

export function MessageThread({
  thread,
  agent,
  agentsUnavailable = false,
  expanded,
  onPickAgent,
  onStartNew,
  artifacts,
  onOpenArtifact,
  onLiveArtifacts,
  onWorkingChange,
  onNavigate,
  openingQuestion,
  onOpeningQuestionSent,
  agentAccent = false,
  page,
  pageRequest,
  onPageRequestSent,
}: {
  thread: AssistantThread;
  agent: AgentChoice | null;
  /** The list of chat agents could not be read, so a missing agent is unknown, not disabled. */
  agentsUnavailable?: boolean;
  expanded: boolean;
  onPickAgent?: () => void;
  /** Starts a fresh conversation with the same agent; offered when this one is full. */
  onStartNew?: () => void;
  /** What the conversation produced, when the surface has a pane to open it in. */
  artifacts?: AssistantArtifact[];
  onOpenArtifact?: (id: string) => void;
  /**
   * Told what a streaming turn has produced so far, each time that changes:
   * an artifact landing, one revised, or one withdrawn because a later read
   * folded it into a table. The newest is last.
   */
  onLiveArtifacts?: (artifacts: readonly AssistantArtifactEvent[]) => void;
  /** Told while a turn is running, for surfaces that show it outside the thread. */
  onWorkingChange?: (working: boolean) => void;
  /**
   * Told just before the app follows a page the assistant opened, so a
   * surface the move takes away can hand the conversation on first.
   */
  onNavigate?: () => void;
  /** A question asked before this thread existed; sent once, as its first message. */
  openingQuestion?: string;
  onOpeningQuestionSent?: () => void;
  /** Washes the thread in the agent's accent, so it reads as that agent's work. */
  agentAccent?: boolean;
  /** Binds the conversation to the page it belongs to; see PageBinding. */
  page?: PageBinding;
  /**
   * Something the page asks for the person, such as explaining the formula on
   * screen, sent once the conversation is free to take it.
   */
  pageRequest?: PageRequest | null;
  onPageRequestSent?: (key: string) => void;
}) {
  const model = useThreadModel({
    thread,
    agent,
    agentsUnavailable,
    artifacts,
    onLiveArtifacts,
    onWorkingChange,
    onNavigate,
    openingQuestion,
    onOpeningQuestionSent,
    page,
    pageRequest,
    onPageRequestSent,
  });
  const {
    t,

    dismissSuggestion,
    draft,
    onDraftChange,
    timezone,
    history,
    messages,
    providers,
    providerId,
    setProviderId,

    switchNotice,
    plansByMessage,
    loosePlans,
    proposalsByMessage,
    looseProposals,
    queue,
    current,
    deferAll,
    resumeAll,
    decided,
    entries,
    placements,
    answerIds,
    sourcesByMessage,
    latestUserSequence,
    readOnly,
    pageContext,
    contextIncluded,
    setContextIncluded,
    turn,
    isActive,
    send,

    stop,
    dismiss,
    retry,
    artifactsByMessage,
    answer,
    showDock,

    block,
    isEmpty,
    suggestions,
    composerContext,
    followUpDecision,

    canTell,
  } = model;

  // What the thread held when it was opened. Only a message numbered past it
  // is an arrival to this reader, and only arrivals rise into place: a page
  // of older history or a row scrolling back into the window does not.
  const openedAt = useRef<number | null>(null);
  // The rows on screen when the thread opens rise into place once, one after
  // another; a row scrolling back into the window later does not, and nor
  // does a page of older history. Which rows have had their rise is kept by
  // message id, so a window that redraws a row never repeats it.
  const [openingDelays, setOpeningDelays] = useState<ReadonlyMap<string, number> | null>(null);
  const risen = useRef(new Set<string>());
  if (openedAt.current === null && !history.isLoading) {
    openedAt.current = highestSequence(messages);
  }
  if (openingDelays === null && !history.isLoading) {
    setOpeningDelays(openingStagger(messages.map((message) => message.id)));
  }
  const arrivals = useMemo(
    () => arrivedSince(openedAt.current ?? Number.POSITIVE_INFINITY, messages),
    [messages],
  );
  // Day markers are relative to today, and today changes while a thread is
  // open; the clock moves only when the reader's date does.
  const now = useCalendarNow(timezone);

  // The composer floats over the bottom of the thread, so the last message has
  // to be padded clear of it and the jump-to-latest button lifted above it. The
  // textarea grows to eight rows, which is why this is measured, not a constant.
  const [composerHeight, setComposerHeight] = useState(0);
  const composerObserver = useRef<ResizeObserver | null>(null);
  // The read-only notice, the approval box and the composer are different
  // elements, and one leaves on its own motion while the next arrives; the
  // one on screen is the one measured, whichever element it is.
  const composerRef = useCallback((element: HTMLDivElement | null) => {
    composerObserver.current?.disconnect();
    composerObserver.current = null;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(([entry]) => {
      setComposerHeight(entry.target.getBoundingClientRect().height);
    });
    observer.observe(element);
    composerObserver.current = observer;
  }, []);
  useEffect(() => () => composerObserver.current?.disconnect(), []);
  // Every row is a closure over its entry, keyed by the message it shows, so
  // the window can measure and place it without knowing what it is. The saved
  // rows never depend on the reply in progress: every token changes the turn,
  // and a saved row whose closure is the same one is not drawn again.
  const savedRows = useMemo<VirtualThreadRow[]>(() => {
    const list: VirtualThreadRow[] = withDayMarkers(entries, now, timezone).map((item) => {
      if (item.kind === "day") {
        return {
          key: item.key,
          render: () => <DayDivider at={item.at} daysAgo={item.daysAgo} />,
        };
      }
      const { entry } = item;
      const arrived = arrivals.has(entry.message.id);
      const delay = openingDelays?.get(entry.message.id);
      return {
        key: entry.message.id,
        render: () => (
          <RiseOnce
            id={entry.message.id}
            rise={arrived || delay !== undefined}
            delay={delay ?? 0}
            risen={risen.current}
          >
            {/* A scheduled request reads as what the person wrote; its card
                and controls are the Desk's. */}
            {entry.kind === "user" || entry.kind === "schedule" ? (
              <UserTurn
                content={entry.message.content}
                sentAt={entry.message.createdAt}
                pageContext={entry.message.pageContext}
                attachments={entry.message.attachments}
                mentions={entry.message.mentions}
                onResend={
                  !readOnly &&
                  entry.kind === "user" &&
                  entry.message.sequence === latestUserSequence
                    ? () => void send(entry.message.content, undefined, providerId)
                    : undefined
                }
              />
            ) : entry.kind === "decision" ? (
              <DecisionNote content={entry.message.content} at={entry.message.createdAt} />
            ) : entry.kind === "declined" ? (
              <DeclinedTurn content={entry.message.content} sentAt={entry.message.createdAt} />
            ) : entry.kind === "refusal" ? (
              <RefusalNotice message={entry.message.content} />
            ) : entry.kind === "compaction" ? (
              <CompactionNote
                summarized={entry.message.compaction?.summarized ?? 0}
                auto={entry.message.compaction?.auto ?? false}
              />
            ) : (
              <AssistantEntry
                entry={entry}
                placement={placements.get(entry.message.id)}
                proposals={proposalsByMessage.get(entry.message.id) ?? []}
                plans={plansByMessage.get(entry.message.id) ?? []}
                artifacts={artifactsByMessage.get(entry.message.id) ?? []}
                latestUserSequence={latestUserSequence}
                onAnswer={answer}
                onOpenArtifact={onOpenArtifact}
                ratable={answerIds.has(entry.message.id)}
                sources={sourcesByMessage.get(entry.message.id)?.sources}
                listsSources={sourcesByMessage.get(entry.message.id)?.answer}
                threadId={thread.id}
              />
            )}
          </RiseOnce>
        ),
      };
    });

    // A proposal whose turn is no longer in the visible thread is shown here
    // rather than dropped: a pending change nobody can see is worse than one
    // shown out of position.
    for (const group of loosePlans) {
      list.push({
        key: `plan-${group.plan.id}`,
        render: () => <PlanRecord plan={group.plan} steps={group.steps} threadId={thread.id} />,
      });
    }
    for (const proposal of looseProposals) {
      list.push({
        key: `proposal-${proposal.id}`,
        render: () => <ProposalRecord proposal={proposal} threadId={thread.id} />,
      });
    }

    return list;
  }, [
    answer,
    answerIds,
    arrivals,
    artifactsByMessage,
    entries,
    latestUserSequence,
    loosePlans,
    looseProposals,
    now,
    onOpenArtifact,
    openingDelays,
    placements,
    sourcesByMessage,
    plansByMessage,
    proposalsByMessage,
    providerId,
    readOnly,
    send,
    thread.id,
    timezone,
  ]);

  const rows = useMemo<VirtualThreadRow[]>(() => {
    if (!turn) {
      return savedRows;
    }

    return [
      ...savedRows,
      {
        key: "turn-in-progress",
        render: () => (
          <div className="animate-rise">
            <StreamingTurn
              turn={turn}
              onRetry={readOnly ? undefined : retry}
              onDismiss={dismiss}
              onAnswer={answer}
              onOpenArtifact={onOpenArtifact}
            />
          </div>
        ),
      },
    ];
  }, [answer, dismiss, onOpenArtifact, readOnly, retry, savedRows, turn]);
  const pill =
    queue.length > 0 && current === null ? (
      <DeferredDecisionsPill count={queue.length} onReopen={resumeAll} />
    ) : null;

  const body = (
    <div data-slot="assistant-thread" className="relative flex min-h-0 flex-1 flex-col">
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
          jumpOffset={Math.max(
            0,
            composerHeight - (expanded ? COMPOSER_FADE : COMPOSER_FADE_COMPACT),
          )}
          // The gutter on the left sits outside the scroll element so the
          // scrollbar stays at the edge; the one on the right is the column's own.
          className={expanded ? "pl-4 lg:pl-6" : "pl-3"}
          contentClassName={expanded ? "max-w-3xl pt-5 pr-4 lg:pr-6" : "pt-4 pr-2"}
          rowClassName={expanded ? "pb-5" : "pb-4"}
        />
      )}

      {/* The approval box leaves on its own motion while the composer takes
          its place beneath it; the three stand-ins share one slot. */}
      <AnimatePresence>
        {showDock && current ? (
          <ApprovalDock
            key="dock"
            ref={composerRef}
            threadId={thread.id}
            entry={current.entry}
            position={current.index + 1}
            total={queue.length}
            compact={!expanded}
            canTell={canTell}
            onDefer={deferAll}
            onDecided={decided}
          />
        ) : block === "read-only" ? (
          <ReadOnlyThreadNotice
            key="read-only"
            ref={composerRef}
            reason={thread.cannotContinueReason}
            compact={!expanded}
            notice={pill}
          />
        ) : (
          <Composer
            key="composer"
            ref={composerRef}
            onSend={(content, payload) => {
              composerContext.clear();
              void send(content, undefined, providerId, payload);
            }}
            onStop={stop}
            active={isActive}
            disabled={block !== null}
            disabledReason={
              block === "full"
                ? t("This conversation is full. Start a new one to continue.")
                : block === "agents-unavailable"
                  ? t(
                      "The agents could not be loaded, so nothing can be sent yet. Refresh to try again.",
                    )
                  : t("This agent has been disabled, so the conversation cannot continue.")
            }
            notice={
              <>
                {pill}
                {history.length.state !== "open" ? (
                  <ThreadLengthNotice
                    state={history.length.state}
                    total={history.total}
                    limit={history.limit}
                    onStartNew={onStartNew}
                  />
                ) : switchNotice ? (
                  <ModelSwitchNotice notice={switchNotice} />
                ) : null}
              </>
            }
            placeholder={
              agent
                ? t("Ask {0}…", agent.name)
                : t("Ask about a shipment, a driver, or how to do something…")
            }
            agent={agent}
            onPickAgent={onPickAgent}
            pageContext={pageContext}
            contextIncluded={contextIncluded}
            onToggleContext={
              page === undefined ? () => setContextIncluded((value) => !value) : undefined
            }
            providers={providers}
            providerId={providerId}
            onPickProvider={setProviderId}
            suggestions={suggestions}
            attachments={composerContext.attachments}
            onAttachFiles={composerContext.attachFiles}
            onRemoveAttachment={composerContext.removeAttachment}
            mentions={composerContext.mentions}
            onMentionsChange={composerContext.setMentions}
            onSearchMentions={composerContext.searchMentions}
            draft={draft}
            onDraftChange={onDraftChange}
            compact={!expanded}
          />
        )}
      </AnimatePresence>
    </div>
  );

  return (
    <AssistantAgentProvider agent={agent} delegates={agent?.delegates}>
      <ArtifactOpenerProvider onOpen={onOpenArtifact}>
        <DecisionFollowUpProvider value={followUpDecision}>
          {agentAccent ? (
            <AgentGutter agent={agent} working={isActive}>
              {body}
            </AgentGutter>
          ) : (
            body
          )}
        </DecisionFollowUpProvider>
      </ArtifactOpenerProvider>
    </AssistantAgentProvider>
  );
}

/**
 * A row rising into place, once. The rise is decided on the row's first
 * draw and remembered by id, so a window that redraws the row later — the
 * reader scrolling away and back — finds it already risen and leaves it.
 */
function RiseOnce({
  id,
  rise,
  delay,
  risen,
  children,
}: {
  id: string;
  rise: boolean;
  delay: number;
  risen: Set<string>;
  children: ReactNode;
}) {
  const [animate] = useState(() => rise && !risen.has(id));
  useEffect(() => {
    if (animate) {
      risen.add(id);
    }
  }, [animate, id, risen]);

  return (
    <div
      className={cn(animate && "animate-rise")}
      style={animate && delay > 0 ? { animationDelay: `${delay}ms` } : undefined}
    >
      {children}
    </div>
  );
}

/** When each of the rows a thread opens on rises, by message id: the last few, top down. */
export function openingStagger(ids: readonly string[]): ReadonlyMap<string, number> {
  const delays = new Map<string, number>();
  const first = Math.max(0, ids.length - OPENING_STAGGER_ROWS);
  for (let index = first; index < ids.length; index += 1) {
    delays.set(ids[index], (index - first) * OPENING_STAGGER_MS);
  }
  for (let index = 0; index < first; index += 1) {
    delays.set(ids[index], 0);
  }

  return delays;
}

/**
 * The first thing a reader sees in a new conversation: the agent's mark in
 * its own light, what the agent is for, what it can see, and a few
 * questions it is good at. The questions are chips that rise one after
 * another, and closing one is a decision that is remembered.
 */
function EmptyThread({
  agent,
  suggestions,
  pageContext,
  onPick,
  onDismiss,
}: {
  agent: AgentChoice | null;
  suggestions: readonly Suggestion[];
  pageContext: AssistantPageContext | null;
  onPick: (prompt: string) => void;
  onDismiss: (prompt: string) => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <div className="mx-auto flex w-full max-w-xl flex-1 flex-col items-center justify-center gap-5 px-4 py-8">
      <m.div
        initial={reduceMotion ? false : { opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.24, ease: EASE_SETTLE }}
        className="flex flex-col items-center gap-3 text-center"
      >
        {/* The mark sits at the head of its own light, the way it does at
            the head of a thread, so the room reads as this agent's before a
            word is said. */}
        <span
          aria-hidden
          className="ui-agent-glow flex h-16 w-40 items-start justify-center"
          style={
            {
              "--agent-accent": agentSpineColor(agent),
              "--agent-glow-rest": 0.9,
              "--agent-glow-extent": "100%",
            } as CSSProperties
          }
        >
          <AgentAvatar size="xl" />
        </span>
        <h2 className="text-base font-semibold">
          {agent ? t("What do you need from {0}?", agent.name) : t("Start a conversation")}
        </h2>
        <p className="text-muted-foreground max-w-md text-sm leading-relaxed">
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
        <AgentStarters
          agentKey={agent?.id ?? ""}
          suggestions={suggestions}
          onPick={(suggestion) => onPick(suggestion.prompt)}
          onDismiss={(suggestion) => onDismiss(suggestion.prompt)}
          className="justify-center"
        />
      )}
    </div>
  );
}

/**
 * Said at the moment a person picks a different model for a conversation
 * that already has replies: the new model starts from nothing and reads the
 * whole thread before it answers.
 */
function ModelSwitchNotice({ notice }: { notice: ModelSwitchNoticeValue }) {
  const t = useT();

  return (
    <div className="text-muted-foreground flex items-center gap-2 px-1 text-xs">
      <InfoCircleIcon className="text-info size-3.5 shrink-0" />
      <span>
        {notice.to
          ? t(
              "Switching to {0}. It will read the whole conversation again before answering, which can take longer and cost more.",
              notice.to,
            )
          : t(
              "Switching to automatic model choice. The next model will read the whole conversation again before answering, which can take longer and cost more.",
            )}
      </span>
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
