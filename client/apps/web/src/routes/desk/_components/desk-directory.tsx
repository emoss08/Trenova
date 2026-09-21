import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  HomeIcon,
  InboxIcon,
  MessageSquareIcon,
  PinIcon,
  PlusIcon,
  SearchIcon,
  Trash2Icon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { NavLink } from "react-router";
import { groupDeskThreads, matchesThreadSearch, type DeskThreadGroup } from "./desk-threads";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

export type DeskDirectoryProps = {
  threads: AssistantThread[];
  agents: AgentDefinitionRow[];
  activeThreadId: string | null;
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string) => void;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
  /** Called when a row is followed, so the surface holding this can close. */
  onNavigate?: () => void;
  className?: string;
};

/**
 * Everywhere the Desk can go: home, the decisions queue, a new conversation
 * with any agent, and every conversation there is.
 *
 * This used to be a 260px column nailed to the left edge, and it was the
 * wrong shape for what it holds. A person picks a conversation perhaps twice
 * an hour and then reads and writes in it for the rest of the hour, so the
 * list was renting a permanent tenth of the window to answer a question
 * that is asked twice. It is a switcher, so it lives behind one now and the
 * width goes to the work.
 */
export function DeskDirectory({
  threads,
  agents,
  activeThreadId,
  isLoading,
  isStarting,
  onStart,
  onDelete,
  onTogglePin,
  onNavigate,
  className,
}: DeskDirectoryProps) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [now] = useState(nowInSeconds);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const decisions = attention?.agentDecisions ?? 0;

  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);
  const agentNames = useMemo(
    () => new Map(agents.map((agent) => [agent.id, agent.name])),
    [agents],
  );
  const groups = useMemo(
    () =>
      groupDeskThreads(
        threads.filter((thread) =>
          matchesThreadSearch(thread, agentNames.get(thread.agentDefinitionId) ?? "", query),
        ),
        agentNames,
      ),
    [agentNames, query, threads],
  );

  return (
    <div className={cn("flex min-h-0 w-full min-w-0 flex-col", className)}>
      <div className="border-border/60 flex items-center gap-1.5 border-b px-2 py-2">
        <Input
          autoFocus
          inputContainerClassName="w-full"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search conversations")}
          className="h-8 text-sm"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          aria-label={t("Search conversations")}
        />
      </div>

      <div className="flex flex-col gap-0.5 p-2">
        <DirectoryLink to="/desk" end icon={HomeIcon} label={t("Today")} onNavigate={onNavigate} />
        {canDecide && (
          <DirectoryLink
            to="/desk/decisions"
            icon={InboxIcon}
            label={t("Decisions")}
            onNavigate={onNavigate}
            trailing={
              decisions > 0 ? (
                <Badge variant="warning" className="h-4 px-1.5 text-2xs tabular-nums">
                  {decisions > 99 ? "99+" : decisions}
                </Badge>
              ) : null
            }
          />
        )}
      </div>

      <div className="border-border/60 border-t px-2 py-2">
        <AgentPicker agents={agents} disabled={isStarting} onStart={onStart} />
      </div>

      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="max-h-[min(26rem,50vh)]"
        maskHeight={16}
      >
        {isLoading ? (
          <div className="space-y-2 p-3">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
          </div>
        ) : groups.length === 0 ? (
          <div className="text-muted-foreground flex flex-col items-center gap-2 px-4 py-10 text-center text-xs">
            <MessageSquareIcon className="size-5" />
            {query.trim() === ""
              ? t("No conversations yet. Start one with an agent.")
              : t("No conversations match that search.")}
          </div>
        ) : (
          <div className="flex flex-col gap-3 p-2">
            {groups.map((group) => (
              <DirectoryGroup
                key={group.kind === "pinned" ? "pinned" : `agent-${group.agentId}`}
                group={group}
                agentsById={agentsById}
                activeThreadId={activeThreadId}
                now={now}
                onDelete={onDelete}
                onTogglePin={onTogglePin}
                onNavigate={onNavigate}
              />
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}

function DirectoryLink({
  to,
  end,
  icon: Icon,
  label,
  trailing,
  onNavigate,
}: {
  to: string;
  end?: boolean;
  icon: typeof HomeIcon;
  label: string;
  trailing?: React.ReactNode;
  onNavigate?: () => void;
}) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onNavigate}
      className={({ isActive }) =>
        cn(
          "ui-focus-ring flex h-8 items-center gap-2 rounded-md px-2 text-sm transition-colors",
          isActive ? "bg-surface-selected text-foreground font-medium" : "hover:bg-surface-hover",
        )
      }
    >
      <Icon className="text-muted-foreground size-4 shrink-0" />
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {trailing}
    </NavLink>
  );
}

/**
 * Which agent to talk to. Shown open rather than behind a second menu: from
 * inside a switcher, one more click to reach the thing the switcher exists
 * to start is a click too many.
 */
function AgentPicker({
  agents,
  disabled,
  onStart,
}: {
  agents: AgentDefinitionRow[];
  disabled: boolean;
  onStart: (agentId: string) => void;
}) {
  const t = useT();

  if (agents.length === 0) {
    return (
      <p className="text-muted-foreground px-2 py-1.5 text-xs">
        {t("No agents are available to talk to yet.")}
      </p>
    );
  }

  return (
    <>
      <p className="text-muted-foreground flex items-center gap-1.5 px-2 pb-1.5 text-xs">
        <PlusIcon className="size-3" />
        {t("Start a conversation with")}
      </p>
      <div className="flex flex-wrap gap-1">
        {agents.map((agent) => (
          <button
            key={agent.id}
            type="button"
            disabled={disabled}
            onClick={() => onStart(agent.id)}
            className={cn(
              "hover:bg-surface-hover ui-focus-ring flex items-center gap-1.5 rounded-full py-1 pr-2.5 pl-1",
              "ring-foreground/10 text-sm ring-1 transition-colors disabled:opacity-50",
            )}
          >
            <AgentTile agent={agent} size="xs" />
            <span className="max-w-40 truncate">{agent.name}</span>
          </button>
        ))}
      </div>
    </>
  );
}

function DirectoryGroup({
  group,
  agentsById,
  activeThreadId,
  now,
  onDelete,
  onTogglePin,
  onNavigate,
}: {
  group: DeskThreadGroup;
  agentsById: Map<string, AgentDefinitionRow>;
  activeThreadId: string | null;
  now: number;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
  onNavigate?: () => void;
}) {
  const t = useT();
  const agent = group.agentId ? (agentsById.get(group.agentId) ?? null) : null;

  return (
    <section className="flex flex-col gap-0.5">
      <h3 className="text-muted-foreground flex items-center gap-1.5 px-2 pb-1 text-xs font-medium">
        {group.kind === "pinned" ? (
          <>
            <PinIcon className="size-3" />
            {t("Pinned")}
          </>
        ) : (
          <>
            <AgentTile agent={agent} size="xs" />
            <span className="truncate">{group.label || t("Agent unavailable")}</span>
          </>
        )}
      </h3>
      {group.threads.map((thread) => (
        <DirectoryRow
          key={thread.id}
          thread={thread}
          showAgent={group.kind === "pinned"}
          agentName={agentsById.get(thread.agentDefinitionId)?.name}
          active={thread.id === activeThreadId}
          now={now}
          onDelete={() => onDelete(thread)}
          onTogglePin={() => onTogglePin(thread)}
          onNavigate={onNavigate}
        />
      ))}
    </section>
  );
}

function DirectoryRow({
  thread,
  showAgent,
  agentName,
  active,
  now,
  onDelete,
  onTogglePin,
  onNavigate,
}: {
  thread: AssistantThread;
  showAgent: boolean;
  agentName: string | undefined;
  active: boolean;
  now: number;
  onDelete: () => void;
  onTogglePin: () => void;
  onNavigate?: () => void;
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
      <NavLink
        to={`/desk/t/${thread.id}`}
        data-thread-row
        onClick={onNavigate}
        className="ui-focus-ring min-w-0 flex-1 rounded-md py-1.5 text-left"
        aria-current={active ? "page" : undefined}
      >
        <span className="block truncate text-sm">{thread.title || t("Untitled conversation")}</span>
        <span className="text-muted-foreground block truncate text-xs">
          {showAgent && agentName ? `${agentName} · ` : ""}
          {formatSecondsAgo(now - touched)}
        </span>
      </NavLink>
      <Button
        variant="ghost"
        size="icon-xs"
        aria-label={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
        aria-pressed={thread.pinned}
        className={cn(
          "transition-opacity focus-visible:opacity-100",
          thread.pinned
            ? "text-foreground"
            : "text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100",
        )}
        onClick={onTogglePin}
      >
        <PinIcon className="size-3.5" />
      </Button>
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
