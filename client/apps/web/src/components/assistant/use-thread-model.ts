import { answerMessageIds } from "@/components/ai-feedback/feedback-targets";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  approvalQueue,
  currentEntry,
  focusKeys,
  holdsFocus,
  type ApprovalEntry,
} from "./approval-queue";
import type { ComposerPayload } from "./composer";
import { decisionRequestsFromSteps } from "./decision-requests";
import { stepsFromSegments } from "./activity";
import { useFollowNavigation } from "./follow-navigation";
import { useApplyDraftEdits } from "./page-draft-edits";
import { modelSwitchNotice } from "./model-switch";
import { groupPlans } from "./plan-state";
import { decidedSignature, groupProposalsByMessage, pollIntervalFor } from "./proposal-state";
import { agentSuggestions } from "./suggestions";
import { composerBlock, shouldSendOpeningQuestion } from "./thread-guard";
import { delegatedOwners, groupThread, turnPlacements } from "./thread-view";
import { useLiveThreadIds } from "./use-active-turns";
import { useAssistantTurn } from "./use-assistant-turn";
import { useComposerContext } from "./use-composer-context";
import { usePageContext } from "./use-page-context";
import { useThreadHistory } from "./use-thread-history";
import { replyWebSources } from "./web-sources";

/** A stable empty list, so a thread with no live turn does not re-run the follower each render. */
const NO_ARTIFACTS: readonly AssistantArtifactEvent[] = [];

/** A stable empty list, so a thread given no artifacts keeps its rows between renders. */
const NO_SAVED_ARTIFACTS: AssistantArtifact[] = [];

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

export type ThreadModelOptions = {
  thread: AssistantThread;
  agent: AgentChoice | null;
  agentsUnavailable?: boolean;
  artifacts?: AssistantArtifact[];
  onLiveArtifacts?: (artifacts: readonly AssistantArtifactEvent[]) => void;
  onWorkingChange?: (working: boolean) => void;
  onNavigate?: () => void;
  openingQuestion?: string;
  onOpeningQuestionSent?: () => void;
  /** Keeps the opening question back, while the files it carries finish uploading. */
  openingHold?: boolean;
  /** What the opening question carries besides its words. */
  openingPayload?: ComposerPayload;
  /**
   * Where the page context comes from, in place of the page the person is on:
   * the Desk is a page of its own, so it sends the page they came from.
   */
  pageContextSource?: () => AssistantPageContext | null;
  page?: PageBinding;
  pageRequest?: PageRequest | null;
  onPageRequestSent?: (key: string) => void;
};

/**
 * Everything a conversation view needs except how it draws: the history, the
 * reply in progress, what waits on a decision and in what order, the model,
 * the page it can see and the draft. Both the assistant panel and the Desk
 * draw a conversation from this, so the two never disagree about what it holds.
 */
export function useThreadModel({
  thread,
  agent,
  agentsUnavailable = false,
  artifacts = NO_SAVED_ARTIFACTS,
  onLiveArtifacts,
  onWorkingChange,
  onNavigate,
  openingQuestion,
  onOpeningQuestionSent,
  openingHold = false,
  openingPayload,
  pageContextSource,
  page,
  pageRequest,
  onPageRequestSent,
}: ThreadModelOptions) {
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
  // Memoized so the thread's rows, which depend on the groups, are not rebuilt
  // on every render of a reply in progress.
  const {
    byMessage: plansByMessage,
    orphans: loosePlans,
    standalone,
  } = useMemo(
    () => groupPlans(plansQuery.data?.results ?? [], proposalsQuery.data?.results ?? [], messages),
    [messages, plansQuery.data, proposalsQuery.data],
  );
  const { byMessage: proposalsByMessage, orphans: looseProposals } = useMemo(
    () => groupProposalsByMessage(standalone, messages),
    [messages, standalone],
  );

  // What waits on the person is asked at the foot of the thread, in the
  // composer's place, one decision at a time and the oldest first; the
  // transcript keeps a line for each. A decision put off for later leaves a
  // pill above the composer instead, and one the agent asked about is shown
  // first whatever else waits.
  const focus = useAssistantStore((state) => state.decisionFocus[thread.id] ?? null);
  const deferredList = useAssistantStore((state) => state.deferredDecisions);
  const deferDecisions = useAssistantStore((state) => state.deferDecisions);
  const resumeDecisions = useAssistantStore((state) => state.resumeDecisions);
  const focusDecision = useAssistantStore((state) => state.focusDecision);
  const clearDecisionFocus = useAssistantStore((state) => state.clearDecisionFocus);
  const queue = useMemo(
    () => approvalQueue(proposalsQuery.data?.results ?? [], plansQuery.data?.results ?? [], focus),
    [focus, plansQuery.data, proposalsQuery.data],
  );
  const deferred = useMemo(() => new Set(deferredList), [deferredList]);
  const current = useMemo(() => currentEntry(queue, deferred, focus), [deferred, focus, queue]);
  const focusHeld = focus !== null && current !== null && holdsFocus(current.entry, focus);
  useEffect(() => {
    if (focus !== null && !focusHeld && !proposalsQuery.isPending && !plansQuery.isPending) {
      clearDecisionFocus(thread.id);
    }
  }, [
    clearDecisionFocus,
    focus,
    focusHeld,
    plansQuery.isPending,
    proposalsQuery.isPending,
    thread.id,
  ]);
  const deferAll = useCallback(() => {
    deferDecisions(queue.flatMap((entry) => entry.members));
    clearDecisionFocus(thread.id);
  }, [clearDecisionFocus, deferDecisions, queue, thread.id]);
  const resumeAll = useCallback(
    () => resumeDecisions(queue.flatMap((entry) => entry.members)),
    [queue, resumeDecisions],
  );
  const decided = useCallback(
    (entry: ApprovalEntry) => {
      if (focus !== null && holdsFocus(entry, focus)) {
        clearDecisionFocus(thread.id);
      }
    },
    [clearDecisionFocus, focus, thread.id],
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
    if (pageContextSource !== undefined) {
      return pageContextSource();
    }
    return contextIncluded ? getPageContext() : null;
  }, [contextIncluded, getPageContext, pageContextSource, readDraft]);
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

  // What the turn has produced is handed to the pane as it changes, so the
  // table opens while the sentence about it is still arriving, grows as
  // later reads join it, and the cards it replaced leave with it.
  const liveArtifacts = turn?.artifacts ?? NO_ARTIFACTS;
  useEffect(() => {
    if (liveArtifacts.length > 0) {
      onLiveArtifacts?.(liveArtifacts);
    }
  }, [liveArtifacts, onLiveArtifacts]);

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
      openingHold ||
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
    void send(pendingQuestion, undefined, providerId, openingPayload);
  }, [
    history.isLoading,
    messages.length,
    onOpeningQuestionSent,
    openingHold,
    openingPayload,
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

  // While the agent is writing, the composer stays: it holds the stop
  // control, and what the reply is about to say may change the decision.
  const showDock = current !== null && !isActive && !history.isLoading;

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

  // The server starts the turn in which the agent reports a decision, once
  // the change has run, wherever the decision was made. This view only has to
  // pick it up: when the thread opens, when a card in it is decided, and when
  // a proposal or plan in it stops waiting because somebody decided it
  // elsewhere — the Desk's decisions, AI Control, another tab.
  const followUpDecision = useCallback(() => void rejoin(), [rejoin]);

  // The agent asked the person to decide something now: the approval box
  // opens on it, put off or not. Only a call seen arriving moves the box; a
  // reopened thread starts from the oldest decision.
  const liveSteps = useMemo(() => (turn ? stepsFromSegments(turn.segments) : []), [turn]);
  const seenRequests = useRef(new Set<string>());
  useEffect(() => {
    for (const request of decisionRequestsFromSteps(liveSteps)) {
      if (seenRequests.current.has(request.callId)) {
        continue;
      }
      seenRequests.current.add(request.callId);
      const next = { proposalIds: request.proposalIds, planId: request.planId };
      focusDecision(thread.id, next, focusKeys(next));
    }
  }, [focusDecision, liveSteps, thread.id]);

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

  const canTell = !readOnly && block === null;

  return {
    t,
    dismissed,
    dismissSuggestion,
    draft,
    onDraftChange,
    timezone,
    history,
    messages,
    providers,
    /** The models were read; an empty list then means none is set up. */
    providersReady: providersQuery.isSuccess,
    providerId,
    setProviderId,
    proposalsQuery,
    plansQuery,
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
    rejoin,
    stop,
    dismiss,
    retry,
    artifactsByMessage,
    answer,
    showDock,
    threadFull,
    block,
    isEmpty,
    suggestions,
    composerContext,
    followUpDecision,
    liveSteps,
    canTell,
  };
}

export type ThreadModel = ReturnType<typeof useThreadModel>;
