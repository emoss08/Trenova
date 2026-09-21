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
import { m, useReducedMotion } from "motion/react";
import { useCallback, useMemo, useState } from "react";
import { groupThreadsByRecency, type ThreadGroup } from "./thread-grouping";

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

/**
 * The conversations, down the side of the expanded panel: a search, a way to
 * start another, and the list shelved by when each was last touched.
 */
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

  const groups = useMemo(
    () => groupThreadsByRecency(filterThreads(threads, agentsById, query), now),
    [agentsById, now, query, threads],
  );

  return (
    <aside className={cn("border-border flex w-72 shrink-0 flex-col border-r", className)}>
      {/* One row, not a filled button stacked over a field. Starting a
          conversation is the cheapest thing in the panel; a solid block at the
          top of the sidebar would outrank Approve, the one place in this
          surface where a filled button means something. */}
      <div className="flex items-center gap-1 px-2 pt-2 pb-1">
        <Input
          inputContainerClassName="w-full"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search conversations")}
          className="h-7 text-xs"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          aria-label={t("Search conversations")}
        />
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
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

      <ScrollArea className="flex-1" maskVariant="popover" maskHeight={16}>
        {isLoading ? (
          <div className="space-y-2 p-3">
            <Skeleton className="h-11" />
            <Skeleton className="h-11" />
            <Skeleton className="h-11" />
          </div>
        ) : (
          <ThreadList
            groups={groups}
            agentsById={agentsById}
            activeThreadId={activeThreadId}
            now={now}
            emptyText={
              query.trim() === ""
                ? t("No conversations yet. Start one to ask a question.")
                : t("No conversations match that search.")
            }
            onSelect={onSelect}
            onDelete={onDelete}
          />
        )}
      </ScrollArea>
    </aside>
  );
}

/** Threads whose title or agent contains the query, case ignored. */
export function filterThreads(
  threads: readonly AssistantThread[],
  agentsById: Map<string, AgentDefinitionRow>,
  query: string,
): AssistantThread[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return [...threads];
  }

  return threads.filter((thread) => {
    const agentName = agentsById.get(thread.agentDefinitionId)?.name ?? "";
    return `${thread.title} ${agentName}`.toLowerCase().includes(needle);
  });
}

export type ThreadListProps = {
  groups: ThreadGroup[];
  agentsById: Map<string, AgentDefinitionRow>;
  activeThreadId: string | null;
  now: number;
  emptyText: string;
  onSelect: (id: string) => void;
  onDelete: (thread: AssistantThread) => void;
};

/**
 * The shelved list, shared by the sidebar and the header's history popover.
 *
 * Arrow keys walk the rows and Enter opens one, so the list can be worked
 * from the search box without reaching for the pointer. The rows arrive with
 * a small stagger when the list mounts and never move again.
 */
export function ThreadList({
  groups,
  agentsById,
  activeThreadId,
  now,
  emptyText,
  onSelect,
  onDelete,
}: ThreadListProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  const onKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") {
      return;
    }
    const rows = Array.from(
      event.currentTarget.querySelectorAll<HTMLButtonElement>("[data-thread-row]"),
    );
    if (rows.length === 0) {
      return;
    }
    const current = rows.indexOf(document.activeElement as HTMLButtonElement);
    const step = event.key === "ArrowDown" ? 1 : -1;
    const next = current === -1 ? 0 : (current + step + rows.length) % rows.length;
    event.preventDefault();
    rows[next].focus();
  }, []);

  if (groups.length === 0) {
    return (
      <div className="text-muted-foreground flex flex-col items-center gap-2 px-4 py-10 text-center text-xs">
        <MessageSquareIcon className="size-5" />
        {emptyText}
      </div>
    );
  }

  let position = 0;

  return (
    <div className="flex flex-col gap-3 p-2" onKeyDown={onKeyDown}>
      {groups.map((group) => (
        <section key={group.label} className="flex flex-col gap-0.5">
          <h3 className="text-muted-foreground px-2 pb-1 text-xs font-medium">{t(group.label)}</h3>
          {group.threads.map((thread) => {
            const index = position;
            position += 1;
            return (
              <m.div
                key={thread.id}
                initial={reduceMotion ? false : { opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.18, delay: Math.min(index * 0.015, 0.18) }}
              >
                <ThreadRow
                  thread={thread}
                  agentName={agentsById.get(thread.agentDefinitionId)?.name}
                  active={thread.id === activeThreadId}
                  now={now}
                  onSelect={() => onSelect(thread.id)}
                  onDelete={() => onDelete(thread)}
                />
              </m.div>
            );
          })}
        </section>
      ))}
    </div>
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
        "group flex items-center gap-1 rounded-md pr-1 pl-2 transition-colors",
        active ? "bg-surface-selected" : "hover:bg-surface-hover",
      )}
    >
      <button
        type="button"
        data-thread-row
        className="ui-focus-ring min-w-0 flex-1 rounded-md py-1.5 text-left"
        onClick={onSelect}
        aria-current={active ? "true" : undefined}
      >
        <span className="block truncate text-sm">{thread.title || t("Untitled conversation")}</span>
        <span className="text-muted-foreground block truncate text-xs">
          {agentName ?? t("Agent unavailable")} · {formatSecondsAgo(now - touched)}
        </span>
      </button>
      <Button
        variant="ghost"
        size="icon-xs"
        aria-label={t("Delete conversation")}
        className="text-muted-foreground hover:text-destructive opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
        onClick={onDelete}
      >
        <Trash2Icon className="size-3.5" />
      </Button>
    </div>
  );
}
