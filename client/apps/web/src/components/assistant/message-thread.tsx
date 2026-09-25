import { AssistantAgentProvider } from "@/components/agent-identity/agent-context";
import { answerMessageIds } from "@/components/ai-feedback/feedback-targets";
import { useCalendarNow } from "@/hooks/use-calendar-now";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import type {
  AssistantArtifact,
  AssistantArtifactEvent,
  AssistantPageContext,
  AssistantPlan,
  AssistantProposal,
  AssistantThread,
} from "@/types/assistant";
import type { PageDraft, PageDraftEdit, PageDraftSurface } from "@/types/page-draft";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { ArrowRightIcon, InfoIcon, XIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ArtifactOpenerProvider } from "./artifact-opener";
import { Composer } from "./composer";
import { DecisionFollowUpProvider } from "./decision-follow-up";
import { useFollowNavigation } from "./follow-navigation";
import { useApplyDraftEdits } from "./page-draft-edits";
import {
  AgentAvatar,
  AssistantEntry,
  DayDivider,
  DecisionNote,
  DeclinedTurn,
  PageContextChip,
  RefusalNotice,
  UserTurn,
} from "./message-items";
import {
  modelSwitchNotice,
  type ModelSwitchNotice as ModelSwitchNoticeValue,
} from "./model-switch";
import { PlanCard } from "./plan-card";
import { groupPlans } from "./plan-state";
import { ProposalCard } from "./proposal-card";
import { decidedSignature, groupProposalsByMessage, pollIntervalFor } from "./proposal-state";
import { ReadOnlyThreadNotice } from "./read-only-thread-notice";
import { StreamingTurn } from "./streaming-turn";
import { agentSuggestions, type Suggestion } from "./suggestions";
import { composerBlock, shouldSendOpeningQuestion } from "./thread-guard";
import { arrivedSince, highestSequence, withDayMarkers } from "./thread-rows";
import { delegatedOwners, groupThread, turnPlacements } from "./thread-view";
import { useLiveThreadIds } from "./use-active-turns";
import { useAssistantTurn } from "./use-assistant-turn";
import { useComposerContext } from "./use-composer-context";
import { usePageContext } from "./use-page-context";
import { useThreadHistory } from "./use-thread-history";
import { VirtualThread, type VirtualThreadRow } from "./virtual-thread";
import { AgentGutter } from "./voice/agent-gutter";
import { replyWebSources } from "./web-sources";

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

/** A stable empty list, so a thread with no live turn does not re-run the follower each render. */
const NO_ARTIFACTS: readonly AssistantArtifactEvent[] = [];

/**
 * A conversation that belongs to a page: the import or formula assistant.
 * The page's unsaved work rides on every turn, and the changes the assistant
 * hands back are applied to the page rather than saved.
 */
export type PageBinding = {
  surface: PageDraftSurface;
  readDraft: () => PageDraft | null;
  onDraftEdit: (edit: PageDraftEdit) => void;
};

/** A question the page asks on the person's behalf, sent once per key. */
export type PageRequest = { key: string; text: string };

export function MessageThread({
  thread,
  agent,
  agentsUnavailable = false,
  expanded,
  onPickAgent,
  onStartNew,
  artifacts = [],
  onOpenArtifact,
  onLiveArtifact,
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
  /** Told each artifact a streaming turn announces, as it lands. */
  onLiveArtifact?: (id: string) => void;
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

  const queryClient = useQueryClient();
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
  // Day markers are relative to today, and today changes while a thread is
  // open; the clock moves only when the reader's date does.
  const now = useCalendarNow(timezone);

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

  // While an approval is being carried out the lists are polled, so the
  // card moves from "waiting for it to run" to its outcome without a
  // remount; the moment nothing is running, they are not.
  const proposalsQuery = useQuery({
    ...queries.assistant.proposals(thread.id),
    refetchInterval: (query) =>
      pollIntervalFor(
        query.state.data?.results ?? [],
        queryClient.getQueryData<{ results: AssistantPlan[] }>(
          queries.assistant.plans(thread.id).queryKey,
        )?.results ?? [],
      ),
  });
  const plansQuery = useQuery({
    ...queries.assistant.plans(thread.id),
    refetchInterval: (query) =>
      pollIntervalFor(
        queryClient.getQueryData<{ results: AssistantProposal[] }>(
          queries.assistant.proposals(thread.id).queryKey,
        )?.results ?? [],
        query.state.data?.results ?? [],
      ),
  });

  // Switching models mid-conversation means the new model reads the whole
  // thread again before it answers; the reader is told at the moment they
  // switch rather than by a slower, costlier reply.
  const switchNotice = modelSwitchNotice({
    pickedId: providerId,
    savedId: thread.preferredProviderId ?? "",
    hasReplies: messages.some(
      (message) => message.role === "Assistant" && message.kind !== "Delegated",
    ),
    providers,
  });
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
  // A reply of several steps is headed once and timed from its question.
  const placements = useMemo(() => turnPlacements(entries), [entries]);
  const answerIds = useMemo(() => answerMessageIds(entries), [entries]);
  const sourcesByMessage = useMemo(() => replyWebSources(entries), [entries]);

  // A question the assistant asked is settled by whatever the person said next,
  // whether they clicked one of its options or typed something else entirely.
  // Comparing positions rather than tracking which button was pressed is what
  // makes a reopened thread render the same as a live one.
  const latestUserSequence = useMemo(
    () =>
      messages.reduce(
        (latest, message) =>
          message.role === "User" && message.kind === "Message"
            ? Math.max(latest, message.sequence)
            : latest,
        -1,
      ),
    [messages],
  );

  // The server says whether the reader may still ask this conversation's
  // agent anything. When not, the conversation stays readable and nothing
  // on it offers to send: no composer, no retry, no suggested questions and
  // no answering a question the agent asked.
  const readOnly = !thread.canContinue;

  const getPageContext = usePageContext();
  const [contextIncluded, setContextIncluded] = useState(true);
  const pageContext = getPageContext();
  // A person who drops the page chip means it: the turn is sent without it
  // rather than with a note saying they would rather it were not there. A
  // conversation that belongs to a page always carries the page and its
  // draft: the page is what it is about.
  const readDraft = page?.readDraft;
  const getTurnContext = useCallback(() => {
    if (readDraft !== undefined) {
      const context = getPageContext();
      return context === null ? null : { ...context, draft: readDraft() };
    }
    return contextIncluded ? getPageContext() : null;
  }, [contextIncluded, getPageContext, readDraft]);
  const { turn, isActive, send, rejoin, stop, dismiss, retry } = useAssistantTurn(
    thread.id,
    getTurnContext,
  );

  // The Desk lights its whole header while an agent works, so the state has
  // to leave the thread. Reported on the way down and cleared on unmount,
  // because a light left on for a thread nobody is looking at is a lie about
  // what the room is doing.
  useEffect(() => {
    onWorkingChange?.(isActive);
  }, [isActive, onWorkingChange]);
  useEffect(() => () => onWorkingChange?.(false), [onWorkingChange]);

  // An artifact announced mid-turn is handed to the pane at once, so the
  // table opens while the sentence about it is still arriving.
  const liveArtifactId = turn?.artifacts.at(-1)?.id ?? null;
  useEffect(() => {
    if (liveArtifactId !== null) {
      onLiveArtifact?.(liveArtifactId);
    }
  }, [liveArtifactId, onLiveArtifact]);

  // "Take me there": a page the assistant opened is followed as it arrives.
  useFollowNavigation(turn?.artifacts ?? NO_ARTIFACTS, onNavigate);

  // A change the assistant hands its page is applied as it arrives, once.
  useApplyDraftEdits(turn?.artifacts ?? NO_ARTIFACTS, page?.surface, page?.onDraftEdit);

  // Each turn's artifacts, by the message that produced them or, before the
  // message id was tied on, by the tool call that did. What another agent
  // produced on a task is tied to its own message, which is drawn inside the
  // hand-off rather than as an entry, so it is shown under the reply that
  // handed the task over.
  const artifactsByMessage = useMemo(() => {
    const owners = delegatedOwners(entries);
    const byMessage = new Map<string, AssistantArtifact[]>();
    const byCall = new Map<string, AssistantArtifact>();
    const add = (messageId: string, artifact: AssistantArtifact) =>
      byMessage.set(messageId, [...(byMessage.get(messageId) ?? []), artifact]);
    for (const artifact of artifacts) {
      if (artifact.messageId) {
        add(owners.get(artifact.messageId) ?? artifact.messageId, artifact);
      } else if (artifact.sourceToolCallId !== "") {
        const owner = owners.get(artifact.sourceToolCallId);
        if (owner) {
          add(owner, artifact);
        } else {
          byCall.set(artifact.sourceToolCallId, artifact);
        }
      }
    }
    for (const entry of entries) {
      if (entry.kind !== "assistant") continue;
      for (const exchange of entry.tools) {
        const artifact = byCall.get(exchange.call.id);
        if (artifact) {
          add(entry.message.id, artifact);
        }
      }
    }
    return byMessage;
  }, [artifacts, entries]);

  // A question typed at the Desk's front door arrives here, because the
  // thread it opened did not exist when it was asked. It is sent once, only
  // into a thread that is genuinely empty, and only once history has loaded —
  // otherwise a reload with the question still in hand would ask it twice.
  const openingSent = useRef(false);
  const pendingQuestion = openingQuestion?.trim() ?? "";
  useEffect(() => {
    if (
      readOnly ||
      !shouldSendOpeningQuestion({
        question: pendingQuestion,
        alreadySent: openingSent.current,
        historyLoading: history.isLoading,
        messageCount: messages.length,
      })
    ) {
      return;
    }
    openingSent.current = true;
    onOpeningQuestionSent?.();
    void send(pendingQuestion, undefined, providerId);
  }, [
    history.isLoading,
    messages.length,
    onOpeningQuestionSent,
    pendingQuestion,
    providerId,
    readOnly,
    send,
  ]);

  // An answer to the assistant's question is an ordinary message. Sending it
  // that way is what keeps a clicked answer and a typed one the same thing:
  // nothing new is stored, and the thread reads identically either way.
  const sendAnswer = useCallback(
    (value: string) => void send(value, undefined, providerId),
    [send, providerId],
  );
  const answer = readOnly ? undefined : sendAnswer;

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
    // The read-only notice and the composer are different elements; the one
    // on screen is the one measured.
  }, [readOnly]);

  const threadFull = history.length.state === "full";
  const block = composerBlock({
    agent,
    agentsUnavailable,
    threadFull,
    canContinue: thread.canContinue,
  });
  const isEmpty = !history.isLoading && entries.length === 0 && turn === null;

  // What the page asks for the person waits for the conversation to be free:
  // history loaded, no reply being written, and a composer that could send it.
  const sentPageRequests = useRef(new Set<string>());
  useEffect(() => {
    if (
      !pageRequest ||
      readOnly ||
      block !== null ||
      history.isLoading ||
      isActive ||
      sentPageRequests.current.has(pageRequest.key)
    ) {
      return;
    }
    sentPageRequests.current.add(pageRequest.key);
    onPageRequestSent?.(pageRequest.key);
    void send(pageRequest.text, undefined, providerId);
  }, [
    block,
    history.isLoading,
    isActive,
    onPageRequestSent,
    pageRequest,
    providerId,
    readOnly,
    send,
  ]);
  // The starter questions are listed on an empty thread and behind a slash
  // in the composer at any time; a dismissed one stays dismissed in both.
  // They are the agent's own, from the server, so a Report Builder is never
  // offered "Where is a shipment?". A conversation that can no longer
  // continue offers none.
  const suggestions = useMemo(
    () =>
      agent && !readOnly
        ? agentSuggestions(agent, contextIncluded ? pageContext : null).filter(
            (item) => !dismissed.includes(item.prompt),
          )
        : [],
    [agent, contextIncluded, dismissed, pageContext, readOnly],
  );

  // The files and records a message carries live beside the draft: uploaded
  // to this thread as they are picked, named from the search as they are
  // typed, and sent as ids the server checks against the thread.
  const composerContext = useComposerContext(thread.id);

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
                attachments={entry.message.attachments}
                mentions={entry.message.mentions}
                onResend={
                  !readOnly && entry.message.sequence === latestUserSequence
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
            ) : (
              <AssistantEntry
                entry={entry}
                placement={placements.get(entry.message.id)}
                proposals={proposalsByMessage.get(entry.message.id) ?? []}
                plans={plansByMessage.get(entry.message.id) ?? []}
                artifacts={artifactsByMessage.get(entry.message.id) ?? []}
                threadId={thread.id}
                latestUserSequence={latestUserSequence}
                onAnswer={answer}
                onOpenArtifact={onOpenArtifact}
                ratable={answerIds.has(entry.message.id)}
                sources={sourcesByMessage.get(entry.message.id)?.sources}
                listsSources={sourcesByMessage.get(entry.message.id)?.answer}
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
            <StreamingTurn
              turn={turn}
              onRetry={readOnly ? undefined : retry}
              onDismiss={dismiss}
              onAnswer={answer}
              onOpenArtifact={onOpenArtifact}
            />
          </div>
        ),
      });
    }

    return list;
  }, [
    answer,
    answerIds,
    arrivals,
    artifactsByMessage,
    dismiss,
    entries,
    latestUserSequence,
    loosePlans,
    looseProposals,
    now,
    onOpenArtifact,
    placements,
    sourcesByMessage,
    plansByMessage,
    proposalsByMessage,
    providerId,
    readOnly,
    retry,
    send,
    thread.id,
    timezone,
    turn,
  ]);

  // The server starts the turn in which the agent reports a decision, once
  // the change has run, wherever the decision was made. This view only has to
  // pick it up: when the thread opens, when a card in it is decided, and when
  // a proposal or plan in it stops waiting because somebody decided it
  // elsewhere — the Desk's decisions, AI Control, another tab.
  const followUpDecision = useCallback(() => void rejoin(), [rejoin]);

  useEffect(() => {
    if (!history.isLoading) {
      void rejoin();
    }
  }, [history.isLoading, rejoin]);

  // A reply this view did not start and is not following — asked from another
  // tab or device while this conversation sat open — is picked up when the
  // list of live replies says it has begun. Rejoining a reply already being
  // followed does nothing.
  const liveHere = useLiveThreadIds().has(thread.id);
  useEffect(() => {
    if (liveHere && !history.isLoading) {
      void rejoin();
    }
  }, [history.isLoading, liveHere, rejoin]);

  const decidedKey = decidedSignature(
    proposalsQuery.data?.results ?? [],
    plansQuery.data?.results ?? [],
  );
  const seenDecided = useRef<string | null>(null);
  useEffect(() => {
    if (proposalsQuery.isPending || plansQuery.isPending) {
      return;
    }
    const previous = seenDecided.current;
    seenDecided.current = decidedKey;
    if (previous !== null && previous !== decidedKey) {
      void rejoin();
    }
  }, [decidedKey, plansQuery.isPending, proposalsQuery.isPending, rejoin]);

  const body = (
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
          jumpOffset={Math.max(
            0,
            composerHeight - (expanded ? COMPOSER_FADE : COMPOSER_FADE_COMPACT),
          )}
          className={expanded ? "pl-4" : "pl-3"}
          contentClassName={expanded ? "max-w-3xl pt-5" : "pt-4"}
          rowClassName={expanded ? "pb-5" : "pb-4"}
        />
      )}

      {block === "read-only" ? (
        <ReadOnlyThreadNotice
          ref={composerRef}
          reason={thread.cannotContinueReason}
          compact={!expanded}
        />
      ) : (
        <Composer
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
            history.length.state !== "open" ? (
              <ThreadLengthNotice
                state={history.length.state}
                total={history.total}
                limit={history.limit}
                onStartNew={onStartNew}
              />
            ) : switchNotice ? (
              <ModelSwitchNotice notice={switchNotice} />
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
  agent: AgentChoice | null;
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
 * Said at the moment a person picks a different model for a conversation
 * that already has replies: the new model starts from nothing and reads the
 * whole thread before it answers.
 */
function ModelSwitchNotice({ notice }: { notice: ModelSwitchNoticeValue }) {
  const t = useT();

  return (
    <div className="text-muted-foreground flex items-center gap-2 px-1 text-xs">
      <InfoIcon className="text-info size-3.5 shrink-0" />
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
