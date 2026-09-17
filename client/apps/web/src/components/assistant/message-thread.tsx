import { useT } from "@trenova/shared/i18n/use-t";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@trenova/shared/components/ui/message-scroller";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
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

export function MessageThread({
  thread,
  agent,
  expanded,
  onPickAgent,
}: {
  thread: AssistantThread;
  agent: AgentDefinitionRow | null;
  expanded: boolean;
  onPickAgent?: () => void;
}) {
  const t = useT();
  const dismissed = useAssistantStore((state) => state.dismissedSuggestions);
  const dismissSuggestion = useAssistantStore((state) => state.dismissSuggestion);

  const messagesQuery = useQuery(queries.assistant.messages(thread.id));
  const messageResults = messagesQuery.data?.results;
  const messages = useMemo(() => messageResults ?? [], [messageResults]);

  // Proposals are fetched rather than taken from the send response: they outlive
  // the turn that raised them, so reopening a thread has to show what is still
  // waiting on a decision.
  const proposalsQuery = useQuery(queries.assistant.proposals(thread.id));
  const { byMessage: proposalsByMessage, orphans: looseProposals } = groupProposalsByMessage(
    proposalsQuery.data?.results ?? [],
    messages,
  );

  const entries = useMemo(() => groupThread(messages), [messages]);
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

  const agentUnavailable = agent === null;
  const isEmpty = !messagesQuery.isLoading && entries.length === 0 && turn === null;
  const suggestions = useMemo(
    () =>
      isEmpty && agent
        ? suggestionsFor(agent.template).filter((item) => !dismissed.includes(item.prompt))
        : [],
    [agent, dismissed, isEmpty],
  );

  return (
    <AssistantAgentProvider agent={agent}>
      <div className="flex min-h-0 flex-1 flex-col">
        <MessageScrollerProvider autoScroll defaultScrollPosition="end">
          <MessageScroller className="flex-1">
            <MessageScrollerViewport className={expanded ? "px-4" : "px-3"}>
              <MessageScrollerContent
                className={expanded ? "mx-auto w-full max-w-3xl gap-5 py-5" : "gap-4 py-4"}
              >
                {messagesQuery.isLoading ? (
                  <div className="flex flex-col gap-4">
                    <Skeleton className="ml-auto h-10 w-2/5" />
                    <Skeleton className="h-20 w-3/5" />
                    <Skeleton className="ml-auto h-10 w-1/3" />
                  </div>
                ) : isEmpty ? (
                  <EmptyThread agent={agent} />
                ) : (
                  entries.map((entry) => (
                    <MessageScrollerItem key={entry.message.id} messageId={entry.message.id}>
                      {entry.kind === "user" ? (
                        <UserBubble
                          content={entry.message.content}
                          sentAt={entry.message.createdAt}
                          pageContext={entry.message.pageContext}
                        />
                      ) : entry.kind === "declined" ? (
                        <DeclinedBubble
                          content={entry.message.content}
                          sentAt={entry.message.createdAt}
                        />
                      ) : entry.kind === "refusal" ? (
                        <RefusalNotice message={entry.message.content} />
                      ) : (
                        <AssistantEntry
                          entry={entry}
                          proposals={proposalsByMessage.get(entry.message.id) ?? []}
                          threadId={thread.id}
                        />
                      )}
                    </MessageScrollerItem>
                  ))
                )}

                {/* A proposal whose turn is no longer in the visible thread is shown
                here rather than dropped: a pending change nobody can see is worse
                than one shown out of position. */}
                {looseProposals.map((proposal) => (
                  <MessageScrollerItem key={proposal.id} messageId={proposal.id}>
                    <ProposalCard proposal={proposal} threadId={thread.id} />
                  </MessageScrollerItem>
                ))}

                {turn && (
                  <MessageScrollerItem messageId="turn-in-progress" scrollAnchor>
                    <div className="flex flex-col gap-4">
                      <StreamingTurn turn={turn} onRetry={retry} onDismiss={dismiss} />
                    </div>
                  </MessageScrollerItem>
                )}
              </MessageScrollerContent>
            </MessageScrollerViewport>
            <MessageScrollerButton />
          </MessageScroller>
        </MessageScrollerProvider>

        <Composer
          onSend={(content) => void send(content)}
          onStop={stop}
          active={isActive}
          disabled={agentUnavailable}
          disabledReason={t("This agent has been disabled, so the conversation cannot continue.")}
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
