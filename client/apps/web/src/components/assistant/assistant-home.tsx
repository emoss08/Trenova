import { useAssistantStore } from "@/stores/assistant-store";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { BotIcon, ChevronRightIcon, PlugZapIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { AgentAsk } from "./agent-ask";
import { LiveReplyLabel } from "./live-reply-label";
import { useAskableAgent } from "./use-askable-agent";

type AssistantHomeProps = {
  agents: AgentChoice[];
  threads: AssistantThread[];
  /** Conversations with a reply still being written. */
  liveThreadIds?: ReadonlySet<string>;
  isLoading: boolean;
  isStarting: boolean;
  canManageAgents: boolean;
  onAsk: (agentId: string, question: string) => void;
  onSelectThread: (id: string) => void;
};

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const RECENT_LIMIT = 4;

/**
 * What the panel shows before a conversation is picked.
 *
 * It used to be a card per agent and then a list of recent threads: two
 * directories stacked in a 400px column, with the thing they both lead to —
 * typing a question — always one click away. So it is the question now, with
 * one searchable agent picker inside the box and the chosen agent's own
 * starter questions under it, and the recent conversations reduced to a
 * short list underneath for the times you meant to go back rather than ask.
 */
export function AssistantHome({
  agents,
  threads,
  liveThreadIds,
  isLoading,
  isStarting,
  canManageAgents,
  onAsk,
  onSelectThread,
}: AssistantHomeProps) {
  const t = useT();
  const closeWidget = useAssistantStore((state) => state.closeWidget);
  const [now] = useState(nowInSeconds);
  const askable = useAskableAgent({ threads });

  if ((!isLoading && agents.length === 0) || askable.noneAvailable) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 px-6 py-10 text-center">
        <span className="bg-sunken text-muted-foreground flex size-12 items-center justify-center rounded-lg">
          <BotIcon className="size-6" />
        </span>
        <div className="flex flex-col gap-1">
          <p className="text-sm font-semibold">{t("No agents are available")}</p>
          <p className="text-muted-foreground text-xs">
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
            render={<Link to="/admin/agent-control" onClick={closeWidget} />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open AI control")}
          </Button>
        )}
      </div>
    );
  }

  const recent = threads.slice(0, RECENT_LIMIT);

  return (
    <ScrollArea className="flex-1" maskVariant="popover">
      <div className="flex flex-col gap-5 px-4 py-4">
        <div className="flex flex-col gap-1 px-1 pt-1">
          <p className="text-sm font-semibold">{t("Ask about anything you can see")}</p>
          <p className="text-muted-foreground text-xs leading-relaxed">
            {t("Reads only what you can already see. Changes wait for your approval.")}
          </p>
        </div>

        {askable.agent === null && askable.choices.isError ? (
          <div className="ring-foreground/10 rounded-surface flex items-center gap-3 px-3 py-2.5 ring-1">
            <p className="text-muted-foreground min-w-0 flex-1 text-xs">
              {t("The agents could not be loaded.")}
            </p>
            <Button size="xs" variant="outline" onClick={askable.choices.refetch}>
              {t("Try again")}
            </Button>
          </div>
        ) : isLoading || askable.agent === null ? (
          <Skeleton className="h-32" />
        ) : (
          <AgentAsk
            variant="compact"
            agent={askable.agent}
            onAgentChange={askable.choose}
            recentIds={askable.recency.ids}
            lastUsedAt={askable.recency.lastUsedAt}
            disabled={isStarting}
            onAsk={onAsk}
          />
        )}

        {recent.length > 0 && (
          <section className="flex flex-col gap-2">
            <h3 className="text-muted-foreground text-xs font-medium">
              {t("Pick up where you left off")}
            </h3>
            <div className="flex flex-col gap-0.5">
              {recent.map((thread) => {
                const touched = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;

                return (
                  <button
                    key={thread.id}
                    type="button"
                    onClick={() => onSelectThread(thread.id)}
                    className="hover:bg-surface-hover ui-focus-ring flex items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm">
                        {thread.title || t("Untitled conversation")}
                      </span>
                      <span className="text-muted-foreground block text-xs">
                        {liveThreadIds?.has(thread.id) ? (
                          <LiveReplyLabel />
                        ) : (
                          formatSecondsAgo(now - touched)
                        )}
                      </span>
                    </span>
                    <ChevronRightIcon className="text-muted-foreground size-4 shrink-0" />
                  </button>
                );
              })}
            </div>
          </section>
        )}
      </div>
    </ScrollArea>
  );
}
