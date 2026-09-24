import { AgentTile } from "@/components/agent-identity/agent-tile";
import { LiveReplyLabel } from "@/components/assistant/live-reply-label";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronRightIcon } from "lucide-react";
import { Link } from "react-router";

export type DeskRecentConversationsProps = {
  threads: readonly AssistantThread[];
  agentsById: ReadonlyMap<string, AgentChoice>;
  liveThreadIds: ReadonlySet<string>;
  now: number;
  className?: string;
};

/**
 * The last few conversations, for the times a person opened the Desk to go
 * back rather than to ask.
 *
 * Each row carries the agent's mark, so a column of titles reads as "the
 * billing one, the dispatch one" at a glance, and a conversation whose reply
 * is still being written says so in place of its time.
 */
export function DeskRecentConversations({
  threads,
  agentsById,
  liveThreadIds,
  now,
  className,
}: DeskRecentConversationsProps) {
  const t = useT();

  if (threads.length === 0) {
    return null;
  }

  return (
    <section aria-labelledby="desk-recent-heading" className={cn("flex flex-col gap-2", className)}>
      <h2 id="desk-recent-heading" className="px-2 text-sm font-semibold">
        {t("Where you left off")}
      </h2>
      <ul className="flex flex-col">
        {threads.map((thread, index) => {
          const agent = agentsById.get(thread.agentDefinitionId) ?? null;
          const touched = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
          const live = liveThreadIds.has(thread.id);

          return (
            <li
              key={thread.id}
              className="animate-rise"
              style={{ animationDelay: `${160 + index * 35}ms` }}
            >
              <Link
                to={conversationPath(thread.id)}
                className={cn(
                  "group ui-focus-ring hover:bg-surface-hover flex items-center gap-2.5 rounded-md px-2 py-1.5",
                  "transition-colors",
                )}
              >
                <AgentTile agent={agent} size="sm" />
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-sm">
                    {thread.title || t("Untitled conversation")}
                  </span>
                  <span className="text-muted-foreground flex min-w-0 items-center gap-1 text-xs">
                    {agent && <span className="truncate">{agent.name}</span>}
                    {agent && <span aria-hidden>·</span>}
                    <span className="shrink-0 tabular-nums">
                      {live ? <LiveReplyLabel /> : formatSecondsAgo(now - touched)}
                    </span>
                  </span>
                </span>
                <ChevronRightIcon
                  aria-hidden
                  className={cn(
                    "text-muted-foreground size-3.5 shrink-0 transition-[opacity,translate]",
                    "-translate-x-1 opacity-0 group-hover:translate-x-0 group-hover:opacity-100",
                    "group-focus-visible:translate-x-0 group-focus-visible:opacity-100",
                  )}
                />
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
