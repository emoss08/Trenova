import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@trenova/shared/components/ui/message-scroller";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { queries } from "@/lib/queries";
import type { AgentDefinition, AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Trash2Icon } from "lucide-react";
import { useMemo, useState } from "react";
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

export function MessageThread({
  thread,
  agent,
  onDelete,
}: {
  thread: AssistantThread;
  agent: AgentDefinition | null;
  onDelete: () => void;
}) {
  const t = useT();
  const [seed, setSeed] = useState<string>();

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
  const { turn, isActive, send, stop, dismiss, retry } = useAssistantTurn(thread.id);

  const agentUnavailable = agent === null;
  const isEmpty = !messagesQuery.isLoading && entries.length === 0 && turn === null;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ThreadHeader thread={thread} agent={agent} onDelete={onDelete} />

      <MessageScrollerProvider autoScroll defaultScrollPosition="end">
        <MessageScroller className="flex-1">
          <MessageScrollerViewport className="px-4">
            <MessageScrollerContent className="mx-auto w-full max-w-3xl gap-5 py-5">
              {messagesQuery.isLoading ? (
                <div className="flex flex-col gap-4">
                  <Skeleton className="ml-auto h-10 w-2/5" />
                  <Skeleton className="h-20 w-3/5" />
                  <Skeleton className="ml-auto h-10 w-1/3" />
                </div>
              ) : isEmpty ? (
                <EmptyThread agent={agent} onSuggest={setSeed} />
              ) : (
                entries.map((entry) => (
                  <MessageScrollerItem key={entry.message.id} messageId={entry.message.id}>
                    {entry.kind === "user" ? (
                      <UserBubble
                        content={entry.message.content}
                        sentAt={entry.message.createdAt}
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
                  <div className="flex flex-col gap-5">
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
        seed={seed}
        onSend={(content) => {
          setSeed(undefined);
          void send(content);
        }}
        onStop={stop}
        active={isActive}
        disabled={agentUnavailable}
        disabledReason={t("This agent has been disabled, so the conversation cannot continue.")}
        placeholder={
          agent
            ? t("Message {0}…", agent.name)
            : t("Ask about a shipment, a driver, or how to do something…")
        }
      />
    </div>
  );
}

/**
 * Who the reader is talking to, and what that agent may do. A conversation is
 * with one agent, and its reach is the thing worth knowing before asking.
 */
function ThreadHeader({
  thread,
  agent,
  onDelete,
}: {
  thread: AssistantThread;
  agent: AgentDefinition | null;
  onDelete: () => void;
}) {
  const t = useT();
  const templatesQuery = useQuery(queries.assistant.agentTemplates());
  const templateLabel = templatesQuery.data?.templates.find(
    (item) => item.template === agent?.template,
  )?.label;

  return (
    <div className="border-border flex items-center gap-3 border-b px-4 py-2.5">
      <AgentAvatar className="size-8" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="truncate text-sm font-medium">
            {agent?.name ?? t("Agent unavailable")}
          </span>
          {templateLabel && <Badge variant="outline">{templateLabel}</Badge>}
          {agent && (
            <Badge variant="secondary">
              {agent.toolNames.length === 0
                ? t("Answers only")
                : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
            </Badge>
          )}
        </div>
        <p className="text-muted-foreground truncate text-xs">
          {agent
            ? agent.description ||
              t("Looks records up for you and proposes changes for your approval.")
            : t(
                "This agent was disabled or removed. You can read the conversation but not continue it.",
              )}
        </p>
      </div>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-muted-foreground hover:text-destructive"
              onClick={onDelete}
              aria-label={t("Delete conversation")}
            />
          }
        >
          <Trash2Icon className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{t("Delete conversation")}</TooltipContent>
      </Tooltip>
      <span className="sr-only">{thread.title}</span>
    </div>
  );
}

/**
 * The first thing a reader sees in a new conversation: what this agent is for
 * and three questions it can actually answer. A blank box teaches nothing.
 */
function EmptyThread({
  agent,
  onSuggest,
}: {
  agent: AgentDefinition | null;
  onSuggest: (prompt: string) => void;
}) {
  const t = useT();
  const suggestions = agent ? suggestionsFor(agent.template) : [];

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 py-16 text-center">
      <AgentAvatar className="size-12 rounded-xl [&_svg]:size-6" />
      <div className="flex flex-col gap-1">
        <p className="text-base font-semibold">
          {agent ? t("Talking to {0}", agent.name) : t("Start a conversation")}
        </p>
        <p className="text-muted-foreground max-w-md text-sm">
          {agent?.description ||
            t(
              "Ask about a shipment, a driver, or how to do something in Trenova. The assistant can look records up and propose changes for you to approve.",
            )}
        </p>
      </div>
      {suggestions.length > 0 && (
        <div className="flex max-w-xl flex-wrap justify-center gap-2">
          {suggestions.map((suggestion) => (
            <button
              key={suggestion.prompt}
              type="button"
              onClick={() => onSuggest(suggestion.prompt)}
              className="bg-card text-muted-foreground hover:bg-muted hover:text-foreground rounded-full border px-3 py-1.5 text-xs transition-colors"
            >
              {t(suggestion.label)}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
