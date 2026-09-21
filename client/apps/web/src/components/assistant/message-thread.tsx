import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AssistantAgentProvider } from "@/components/agent-identity/agent-context";
import { Composer } from "./composer";
import {
  AgentAvatar,
  AssistantEntry,
  DeclinedBubble,
  RefusalNotice,
  UserBubble,
} from "./message-items";
import { ProposalCard } from "./proposal-card";
import { groupProposalsByMessage } from "./proposal-state";
import { StreamingTurn } from "./streaming-turn";
import { suggestionsFor } from "./suggestions";
import { groupThread } from "./thread-view";
import { useAssistantTurn } from "./use-assistant-turn";
import { usePageContext } from "./use-page-context";
import { useThreadHistory } from "./use-thread-history";
import { VirtualThread, type VirtualThreadRow } from "./virtual-thread";

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

  const proposalsQuery = useQuery(queries.assistant.proposals(thread.id));
  const { byMessage: proposalsByMessage, orphans: looseProposals } = groupProposalsByMessage(
    proposalsQuery.data?.results ?? [],
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
  const suggestions = useMemo(
    () =>
      isEmpty && agent
        ? suggestionsFor(agent.template).filter((item) => !dismissed.includes(item.prompt))
        : [],
    [agent, dismissed, isEmpty],
  );

  // Every row is a closure over its entry, keyed by the message it shows, so
  // the window can measure and place it without knowing what it is.
  const rows = useMemo<VirtualThreadRow[]>(() => {
    const list: VirtualThreadRow[] = entries.map((entry) => ({
      key: entry.message.id,
      render: () =>
        entry.kind === "user" ? (
          <UserBubble
            content={entry.message.content}
            sentAt={entry.message.createdAt}
            pageContext={entry.message.pageContext}
          />
        ) : entry.kind === "declined" ? (
          <DeclinedBubble content={entry.message.content} sentAt={entry.message.createdAt} />
        ) : entry.kind === "refusal" ? (
          <RefusalNotice message={entry.message.content} />
        ) : (
          <AssistantEntry
            entry={entry}
            proposals={proposalsByMessage.get(entry.message.id) ?? []}
            threadId={thread.id}
            latestUserSequence={latestUserSequence}
            onAnswer={answer}
          />
        ),
    }));

    // A proposal whose turn is no longer in the visible thread is shown here
    // rather than dropped: a pending change nobody can see is worse than one
    // shown out of position.
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
          <StreamingTurn turn={turn} onRetry={retry} onDismiss={dismiss} onAnswer={answer} />
        ),
      });
    }

    return list;
  }, [
    answer,
    dismiss,
    entries,
    latestUserSequence,
    looseProposals,
    proposalsByMessage,
    retry,
    thread.id,
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
            <EmptyThread agent={agent} />
          </div>
        ) : (
          <VirtualThread
            rows={rows}
            hasOlder={history.hasOlder}
            isLoadingOlder={history.isLoadingOlder}
            onLoadOlder={history.loadOlder}
            paddingBottom={composerHeight}
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
          onDismissSuggestion={dismissSuggestion}
          compact={!expanded}
        />
      </div>
    </AssistantAgentProvider>
  );
}

/**
 * The first thing a reader sees in a new conversation: what this agent is for.
 * The opening questions sit on the composer, where they can be sent or closed.
 */
function EmptyThread({ agent }: { agent: AgentDefinitionRow | null }) {
  const t = useT();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-4 py-10 text-center">
      <AgentAvatar size="xl" />
      <div className="flex flex-col gap-1 px-4">
        <p className="text-sm font-semibold">
          {agent ? t("Talking to {0}", agent.name) : t("Start a conversation")}
        </p>
        <p className="text-muted-foreground max-w-sm text-xs">
          {agent?.description ||
            t(
              "Ask about a shipment, a driver, or how to do something in Trenova. The assistant can look records up and propose changes for you to approve.",
            )}
        </p>
      </div>
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
