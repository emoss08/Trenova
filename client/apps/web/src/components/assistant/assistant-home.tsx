import { useAssistantStore } from "@/stores/assistant-store";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { BotIcon, ChevronRightIcon, PlugZapIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { AskBox } from "./ask-box";

type AssistantHomeProps = {
  agents: AgentDefinitionRow[];
  threads: AssistantThread[];
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
 * the agent picker inside the box, and the recent conversations reduced to a
 * short list underneath for the times you meant to go back rather than ask.
 */
export function AssistantHome({
  agents,
  threads,
  isLoading,
  isStarting,
  canManageAgents,
  onAsk,
  onSelectThread,
}: AssistantHomeProps) {
  const t = useT();
  const closeWidget = useAssistantStore((state) => state.closeWidget);
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const [now] = useState(nowInSeconds);

  if (!isLoading && agents.length === 0) {
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

        {isLoading ? (
          <Skeleton className="h-32" />
        ) : (
          <AskBox
            agents={agents}
            defaultAgentId={lastAgentId}
            disabled={isStarting}
            compact
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
                        {formatSecondsAgo(now - touched)}
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
