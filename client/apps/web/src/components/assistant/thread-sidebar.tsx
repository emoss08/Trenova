import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { MessageSquareIcon, PlusIcon, SearchIcon, Trash2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { groupThreadsByRecency } from "./thread-grouping";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

export type ThreadSidebarProps = {
  threads: AssistantThread[];
  agentsById: Map<string, AgentDefinitionRow>;
  activeThreadId: string | null;
  isLoading: boolean;
  canStart: boolean;
  onSelect: (id: string) => void;
  onStart: () => void;
  onDelete: (thread: AssistantThread) => void;
  className?: string;
};

export function ThreadSidebar({
  threads,
  agentsById,
  activeThreadId,
  isLoading,
  canStart,
  onSelect,
  onStart,
  onDelete,
  className,
}: ThreadSidebarProps) {
  const t = useT();
  const [query, setQuery] = useState("");
  // Read once per mount: the shelves are relative to when the list was opened,
  // and a clock read during render would make every render impure.
  const [now] = useState(nowInSeconds);

  const groups = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const visible =
      needle === ""
        ? threads
        : threads.filter((thread) => {
            const agentName = agentsById.get(thread.agentDefinitionId)?.name ?? "";
            return `${thread.title} ${agentName}`.toLowerCase().includes(needle);
          });
    return groupThreadsByRecency(visible, now);
  }, [agentsById, now, query, threads]);

  return (
    <aside
      className={cn("border-border bg-sidebar flex w-72 shrink-0 flex-col border-r", className)}
    >
      {/* One row, not a filled button stacked over a field. Starting a
          conversation is the cheapest thing in the panel; a solid brand block
          at the top of the sidebar outranked Approve, which is the one place in
          this surface where a filled button means something. */}
      <div className="border-border flex items-center gap-1 border-b p-2">
        <Input
          inputContainerClassName="w-full"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search")}
          className="h-7 text-xs"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          aria-label={t("Search conversations")}
        />
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon-sm"
                className="shrink-0"
                onClick={onStart}
                disabled={!canStart}
                aria-label={t("New conversation")}
              />
            }
          >
            <PlusIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent>{t("New conversation")}</TooltipContent>
        </Tooltip>
      </div>

      <ScrollArea className="flex-1" maskVariant="sidebar" maskHeight={16}>
        {isLoading ? (
          <div className="space-y-2 p-3">
            <Skeleton className="h-11" />
            <Skeleton className="h-11" />
            <Skeleton className="h-11" />
          </div>
        ) : groups.length === 0 ? (
          <div className="text-muted-foreground flex flex-col items-center gap-2 px-4 py-10 text-center text-xs">
            <MessageSquareIcon className="size-5" />
            {query.trim() === ""
              ? t("No conversations yet. Start one to ask a question.")
              : t("No conversations match that search.")}
          </div>
        ) : (
          <div className="flex flex-col gap-3 p-2">
            {groups.map((group) => (
              <section key={group.label} className="flex flex-col gap-0.5">
                <h3 className="text-muted-foreground px-2 pb-1 text-xs font-medium">
                  {t(group.label)}
                </h3>
                {group.threads.map((thread) => (
                  <ThreadRow
                    key={thread.id}
                    thread={thread}
                    agentName={agentsById.get(thread.agentDefinitionId)?.name}
                    active={thread.id === activeThreadId}
                    now={now}
                    onSelect={() => onSelect(thread.id)}
                    onDelete={() => onDelete(thread)}
                  />
                ))}
              </section>
            ))}
          </div>
        )}
      </ScrollArea>
    </aside>
  );
}

function ThreadRow({
  thread,
  agentName,
  active,
  now,
  onSelect,
  onDelete,
}: {
  thread: AssistantThread;
  agentName: string | undefined;
  active: boolean;
  now: number;
  onSelect: () => void;
  onDelete: () => void;
}) {
  const t = useT();
  const touched = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;

  return (
    <div
      className={cn(
        "group flex items-center gap-1 rounded-md px-2 py-1.5 transition-colors",
        active ? "bg-muted" : "hover:bg-muted/60",
      )}
    >
      <button
        type="button"
        className="min-w-0 flex-1 text-left"
        onClick={onSelect}
        aria-current={active ? "true" : undefined}
      >
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm">
            {thread.title || t("Untitled conversation")}
          </span>
          <span className="text-muted-foreground block truncate text-xs">
            {agentName ?? t("Agent unavailable")} · {formatSecondsAgo(now - touched)}
          </span>
        </span>
      </button>
      <Button
        variant="ghost"
        size="icon-xxs"
        aria-label={t("Delete conversation")}
        className="text-muted-foreground hover:text-destructive opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
        onClick={onDelete}
      >
        <Trash2Icon className="size-3.5" />
      </Button>
    </div>
  );
}
