import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
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

export type DeskRailProps = {
  threads: AssistantThread[];
  agents: AgentDefinitionRow[];
  activeThreadId: string | null;
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string) => void;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
  className?: string;
};

/**
 * The Desk's left edge: where to go (home, decisions), a way to start a
 * conversation with any agent, a search, and the conversations shelved by
 * pin and by agent. It is the one part of the Desk that stays put while
 * the middle changes.
 */
export function DeskRail({
  threads,
  agents,
  activeThreadId,
  isLoading,
  isStarting,
  onStart,
  onDelete,
  onTogglePin,
  className,
}: DeskRailProps) {
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
    <nav
      aria-label={t("Desk")}
      className={cn("bg-sunken flex min-h-0 w-full min-w-0 flex-col", className)}
    >
      <div className="flex flex-col gap-0.5 px-2 pt-2">
        <RailLink to="/desk" end icon={HomeIcon} label={t("Home")} />
        {canDecide && (
          <RailLink
            to="/desk/decisions"
            icon={InboxIcon}
            label={t("Decisions")}
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

      <div className="flex items-center gap-1 px-2 pt-3 pb-1">
        <Input
          inputContainerClassName="w-full"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search conversations")}
          className="h-7 text-xs"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          aria-label={t("Search conversations")}
        />
        <NewConversationMenu agents={agents} disabled={isStarting} onStart={onStart} />
      </div>

      <ScrollArea className="min-h-0 flex-1" maskHeight={16}>
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
              <RailGroup
                key={group.kind === "pinned" ? "pinned" : `agent-${group.agentId}`}
                group={group}
                agentsById={agentsById}
                activeThreadId={activeThreadId}
                now={now}
                onDelete={onDelete}
                onTogglePin={onTogglePin}
              />
            ))}
          </div>
        )}
      </ScrollArea>
    </nav>
  );
}

function RailLink({
  to,
  end,
  icon: Icon,
  label,
  trailing,
}: {
  to: string;
  end?: boolean;
  icon: typeof HomeIcon;
  label: string;
  trailing?: React.ReactNode;
}) {
  return (
    <NavLink
      to={to}
      end={end}
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

function NewConversationMenu({
  agents,
  disabled,
  onStart,
}: {
  agents: AgentDefinitionRow[];
  disabled: boolean;
  onStart: (agentId: string) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="shrink-0"
                  disabled={disabled || agents.length === 0}
                  aria-label={t("New conversation")}
                />
              }
            />
          }
        >
          <PlusIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{t("New conversation")}</TooltipContent>
      </Tooltip>
      <PopoverContent align="start" className="w-72 p-1.5">
        <p className="text-muted-foreground px-2 py-1 text-xs font-medium">
          {t("Start a conversation with")}
        </p>
        <ScrollArea viewportClassName="max-h-72">
          <div className="flex flex-col gap-0.5">
            {agents.map((agent) => (
              <button
                key={agent.id}
                type="button"
                onClick={() => {
                  setOpen(false);
                  onStart(agent.id);
                }}
                className="hover:bg-surface-hover ui-focus-ring flex items-center gap-2.5 rounded-md px-2 py-1.5 text-left transition-colors"
              >
                <AgentTile agent={agent} size="sm" />
                <span className="min-w-0 flex-1 truncate text-sm">{agent.name}</span>
              </button>
            ))}
          </div>
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
}

function RailGroup({
  group,
  agentsById,
  activeThreadId,
  now,
  onDelete,
  onTogglePin,
}: {
  group: DeskThreadGroup;
  agentsById: Map<string, AgentDefinitionRow>;
  activeThreadId: string | null;
  now: number;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
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
        <RailRow
          key={thread.id}
          thread={thread}
          showAgent={group.kind === "pinned"}
          agentName={agentsById.get(thread.agentDefinitionId)?.name}
          active={thread.id === activeThreadId}
          now={now}
          onDelete={() => onDelete(thread)}
          onTogglePin={() => onTogglePin(thread)}
        />
      ))}
    </section>
  );
}

function RailRow({
  thread,
  showAgent,
  agentName,
  active,
  now,
  onDelete,
  onTogglePin,
}: {
  thread: AssistantThread;
  showAgent: boolean;
  agentName: string | undefined;
  active: boolean;
  now: number;
  onDelete: () => void;
  onTogglePin: () => void;
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
