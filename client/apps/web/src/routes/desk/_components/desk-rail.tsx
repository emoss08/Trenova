import { AgentTile } from "@/components/agent-identity/agent-tile";
import { AgentPicker } from "@/components/assistant/agent-picker";
import { LiveReplyLabel } from "@/components/assistant/live-reply-label";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { RECENT_AGENT_LIMIT } from "@/components/assistant/use-askable-agent";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { agentRecency } from "@/lib/recent-agents";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AssistantThread } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeftIcon,
  HomeIcon,
  InboxIcon,
  MessageSquareIcon,
  PanelLeftCloseIcon,
  PinIcon,
  PlusIcon,
  RadarIcon,
  SearchIcon,
  Trash2Icon,
  TruckIcon,
  XIcon,
} from "lucide-react";
import { useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { Link, NavLink } from "react-router";
import {
  groupDeskThreadsByRecency,
  matchesThreadSearch,
  type DeskShelfKey,
  type DeskThreadShelf,
} from "./desk-threads";

/** The keystroke that folds and unfolds the rail, as it is shown beside the control. */
export const RAIL_SHORTCUT = "⌘B";
/** The same keystroke as `aria-keyshortcuts` reads it, on either modifier. */
export const RAIL_KEYSHORTCUTS = "Meta+B Control+B";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/** Rows arrive a beat apart; past this many the rest land together. */
const ROW_STAGGER_MS = 14;
const ROW_STAGGER_CAP = 12;

export type DeskRailProps = {
  threads: AssistantThread[];
  agents: AgentChoice[];
  activeThreadId: string | null;
  isLoading: boolean;
  /** The conversations could not be read; `onRetry` asks again. */
  listUnavailable: boolean;
  isStarting: boolean;
  onStart: (agentId: string) => void;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
  onRetry: () => void;
  /** Folds the rail away; on a narrow screen, closes the sheet it is in. */
  onCollapse: () => void;
  /** What the fold control is called, with its keystroke shown only where one applies. */
  collapse: { label: string; shortcut?: string };
  /** Called when a row is followed, so a sheet holding this can close. */
  onNavigate?: () => void;
  className?: string;
};

/**
 * The Desk's left rail: where the Desk can go and every conversation there is.
 *
 * It is the conversation's table of contents, which is why it reads top to
 * bottom the way a day does — the way out and the way in first, then the
 * places, then the conversations shelved by when they were last touched,
 * what the person pinned above the calendar. A person scanning for "the one
 * from this morning" finds it under Today; one who remembers the agent reads
 * the mark on each row.
 *
 * It folds to nothing with ⌘B and the width goes to the work, so the room is
 * never paying for a list nobody is reading. The fold is remembered.
 */
export function DeskRail({
  threads,
  agents,
  activeThreadId,
  isLoading,
  listUnavailable,
  isStarting,
  onStart,
  onDelete,
  onTogglePin,
  onRetry,
  onCollapse,
  collapse,
  onNavigate,
  className,
}: DeskRailProps) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [now] = useState(nowInSeconds);
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const { data: watchtowerCounts } = useQuery({
    ...queries.watchtower.counts(),
    enabled: canWatch,
  });
  const decisions = attention?.agentDecisions ?? 0;
  const unseen = watchtowerCounts?.unseen ?? 0;
  const liveThreadIds = useLiveThreadIds();

  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);
  const searching = query.trim() !== "";
  const shelves = useMemo(
    () =>
      groupDeskThreadsByRecency(
        threads.filter((thread) =>
          matchesThreadSearch(thread, agentsById.get(thread.agentDefinitionId)?.name ?? "", query),
        ),
        now,
        { pinnedFirst: true, timezone },
      ),
    [agentsById, now, query, threads, timezone],
  );

  return (
    <div
      data-slot="desk-rail"
      className={cn(
        "bg-desk-rail text-foreground flex h-full w-full min-w-0 flex-col overflow-hidden",
        className,
      )}
    >
      <div className="flex h-12 shrink-0 items-center gap-1 pr-2 pl-3">
        <Link
          to="/desk"
          onClick={onNavigate}
          className="ui-focus-ring flex min-w-0 items-center gap-2 rounded-md py-1 pr-2"
        >
          <span
            aria-hidden
            className="bg-ink text-ink-foreground flex size-6 shrink-0 items-center justify-center rounded-md"
          >
            <TruckIcon className="size-3.5" />
          </span>
          <span className="truncate text-sm font-semibold">{t("Desk")}</span>
        </Link>
        <span className="flex-1" />
        <RailAction label={collapse.label} shortcut={collapse.shortcut} onClick={onCollapse}>
          <PanelLeftCloseIcon className="size-4" />
        </RailAction>
      </div>

      <div className="flex shrink-0 flex-col gap-2 px-3 pt-1 pb-3">
        <NewConversationPicker threads={threads} disabled={isStarting} onStart={onStart} />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search conversations")}
          aria-label={t("Search conversations")}
          inputContainerClassName="w-full"
          className="h-8 text-sm"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          rightElement={
            searching ? (
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label={t("Clear search")}
                onClick={() => setQuery("")}
              >
                <XIcon className="size-3" />
              </Button>
            ) : null
          }
        />
      </div>

      <nav aria-label={t("Desk")} className="flex shrink-0 flex-col gap-0.5 px-2">
        <RailLink to="/desk" end icon={HomeIcon} label={t("Today")} onNavigate={onNavigate} />
        {canWatch && (
          <RailLink
            to="/desk/watchtower"
            icon={RadarIcon}
            label={t("Watchtower")}
            onNavigate={onNavigate}
            trailing={
              unseen > 0 ? (
                <Badge
                  variant={
                    watchtowerCounts && watchtowerCounts.unseenCritical > 0 ? "danger" : "neutral"
                  }
                  className="text-2xs h-4 px-1.5 tabular-nums"
                >
                  {unseen > 99 ? "99+" : unseen}
                </Badge>
              ) : null
            }
          />
        )}
        {canDecide && (
          <RailLink
            to="/desk/decisions"
            icon={InboxIcon}
            label={t("Decisions")}
            onNavigate={onNavigate}
            trailing={
              decisions > 0 ? (
                <Badge variant="warning" className="text-2xs h-4 px-1.5 tabular-nums">
                  {decisions > 99 ? "99+" : decisions}
                </Badge>
              ) : null
            }
          />
        )}
      </nav>

      <ScrollArea className="mt-3 min-h-0 flex-1" maskHeight={0}>
        {isLoading ? (
          <RailListSkeleton />
        ) : listUnavailable ? (
          <RailNotice icon={<MessageSquareIcon className="size-5" />}>
            {t("The conversations could not be loaded.")}
            <Button size="sm" variant="outline" onClick={onRetry}>
              {t("Try again")}
            </Button>
          </RailNotice>
        ) : shelves.length === 0 ? (
          <RailNotice icon={<MessageSquareIcon className="size-5" />}>
            {searching
              ? t("No conversations match that search.")
              : t("No conversations yet. Start one with an agent.")}
          </RailNotice>
        ) : (
          <RailShelves
            shelves={shelves}
            agentsById={agentsById}
            activeThreadId={activeThreadId}
            liveThreadIds={liveThreadIds}
            onDelete={onDelete}
            onTogglePin={onTogglePin}
            onNavigate={onNavigate}
          />
        )}
      </ScrollArea>

      <footer className="border-desk-hairline flex shrink-0 items-center gap-2 border-t px-2 py-2">
        <ResolvedUserAvatar
          userId={user?.id}
          name={user?.name}
          profilePicUrl={user?.profilePicUrl}
          thumbnailUrl={user?.thumbnailUrl}
          className="size-7"
          fallbackClassName="bg-muted text-2xs font-medium text-muted-foreground"
        />
        <span className="grid min-w-0 flex-1 leading-tight">
          <span className="truncate text-sm font-medium">
            {user?.name ?? user?.username ?? t("Signed in")}
          </span>
          {user?.emailAddress && (
            <span className="text-muted-foreground truncate text-xs">{user.emailAddress}</span>
          )}
        </span>
        <Tooltip>
          <TooltipTrigger
            render={
              <Link
                to="/"
                aria-label={t("Back to Trenova")}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "icon-sm" }),
                  "text-muted-foreground hover:text-foreground shrink-0",
                )}
              />
            }
          >
            <ArrowLeftIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent side="top">{t("Back to Trenova")}</TooltipContent>
        </Tooltip>
      </footer>
    </div>
  );
}

function RailAction({
  label,
  shortcut,
  onClick,
  children,
}: {
  label: string;
  /** The keystroke that does the same thing, shown in the tooltip. */
  shortcut?: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={label}
            aria-keyshortcuts={shortcut ? RAIL_KEYSHORTCUTS : undefined}
            className="text-muted-foreground hover:text-foreground shrink-0"
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="bottom" className="flex items-center gap-2">
        {label}
        {shortcut && <Kbd>{shortcut}</Kbd>}
      </TooltipContent>
    </Tooltip>
  );
}

function RailLink({
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
  trailing?: ReactNode;
  onNavigate?: () => void;
}) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onNavigate}
      className={({ isActive }) =>
        cn(
          "ui-focus-ring flex h-8 items-center gap-2.5 rounded-md px-2.5 text-sm transition-colors",
          isActive
            ? "bg-surface-selected text-foreground font-medium"
            : "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
        )
      }
    >
      {({ isActive }) => (
        <>
          <Icon
            className={cn(
              "size-4 shrink-0",
              isActive ? "text-foreground" : "text-muted-foreground",
            )}
          />
          <span className="min-w-0 flex-1 truncate">{label}</span>
          {trailing}
        </>
      )}
    </NavLink>
  );
}

/**
 * Which agent to start a conversation with. One ink control that opens the
 * searchable picker rather than a chip per agent: a rail that listed every
 * agent grew with the organization until the conversations it exists to
 * reach were pushed out of view.
 */
function NewConversationPicker({
  threads,
  disabled,
  onStart,
}: {
  threads: AssistantThread[];
  disabled: boolean;
  onStart: (agentId: string) => void;
}) {
  const t = useT();
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const recency = useMemo(
    () => agentRecency(threads, { limit: RECENT_AGENT_LIMIT, preferId: lastAgentId }),
    [lastAgentId, threads],
  );

  return (
    <AgentPicker
      agent={null}
      recentIds={recency.ids}
      lastUsedAt={recency.lastUsedAt}
      disabled={disabled}
      onSelect={(agent) => onStart(agent.id)}
      trigger={
        <Button type="button" size="lg" className="w-full justify-start px-3 has-[>svg]:px-3">
          <PlusIcon className="size-4" />
          <span className="min-w-0 flex-1 truncate text-left">{t("New conversation")}</span>
        </Button>
      }
    />
  );
}

function RailNotice({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <div className="text-muted-foreground flex flex-col items-center gap-3 px-4 py-10 text-center text-xs text-pretty">
      {icon}
      {children}
    </div>
  );
}

function RailListSkeleton() {
  return (
    <div className="flex flex-col gap-1.5 px-3 py-1" aria-busy>
      <Skeleton className="mb-1 h-3 w-12" />
      {Array.from({ length: 5 }, (_, index) => (
        <Skeleton key={index} className="h-10" />
      ))}
    </div>
  );
}

/** What each shelf is called, as literal strings so the catalog carries them. */
function shelfHeading(t: TranslateFn, key: DeskShelfKey): string {
  switch (key) {
    case "pinned":
      return t("Pinned");
    case "today":
      return t("Today");
    case "yesterday":
      return t("Yesterday");
    case "week":
      return t("Previous 7 days");
    case "month":
      return t("Previous 30 days");
    default:
      return t("Older");
  }
}

function RailShelves({
  shelves,
  agentsById,
  activeThreadId,
  liveThreadIds,
  onDelete,
  onTogglePin,
  onNavigate,
}: {
  shelves: DeskThreadShelf[];
  agentsById: ReadonlyMap<string, AgentChoice>;
  activeThreadId: string | null;
  liveThreadIds: ReadonlySet<string>;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
  onNavigate?: () => void;
}) {
  const t = useT();
  // Where each shelf's first row stands in the whole list, so the stagger
  // runs down the rail rather than restarting at every heading.
  const starts: number[] = [];
  shelves.reduce((count, shelf) => {
    starts.push(count);
    return count + shelf.threads.length;
  }, 0);

  return (
    <div className="flex flex-col gap-4 px-2 pb-3">
      {shelves.map((shelf, shelfIndex) => (
        <section
          key={shelf.key}
          aria-label={shelfHeading(t, shelf.key)}
          className="flex flex-col gap-0.5"
        >
          <h3 className="text-foreground-subtle flex items-center gap-1.5 px-2.5 pb-1 text-xs font-medium">
            {shelf.key === "pinned" && <PinIcon className="size-3" />}
            {shelfHeading(t, shelf.key)}
          </h3>
          {shelf.threads.map((thread, index) => (
            <RailRow
              key={thread.id}
              thread={thread}
              agent={agentsById.get(thread.agentDefinitionId) ?? null}
              active={thread.id === activeThreadId}
              live={liveThreadIds.has(thread.id)}
              style={{
                animationDelay: `${Math.min(starts[shelfIndex] + index, ROW_STAGGER_CAP) * ROW_STAGGER_MS}ms`,
              }}
              onDelete={() => onDelete(thread)}
              onTogglePin={() => onTogglePin(thread)}
              onNavigate={onNavigate}
            />
          ))}
        </section>
      ))}
    </div>
  );
}

function RailRow({
  thread,
  agent,
  active,
  live,
  style,
  onDelete,
  onTogglePin,
  onNavigate,
}: {
  thread: AssistantThread;
  agent: AgentChoice | null;
  active: boolean;
  live: boolean;
  style: CSSProperties;
  onDelete: () => void;
  onTogglePin: () => void;
  onNavigate?: () => void;
}) {
  const t = useT();

  return (
    <div
      data-thread-row
      style={style}
      className={cn(
        "group animate-rise relative flex items-center rounded-md transition-colors",
        active ? "bg-surface-selected" : "hover:bg-surface-hover",
      )}
    >
      <NavLink
        to={conversationPath(thread.id)}
        onClick={onNavigate}
        className="ui-inset-focus-ring flex min-w-0 flex-1 flex-col gap-0.5 rounded-md py-1.5 pr-14 pl-2.5"
      >
        <span className={cn("truncate text-sm", active && "font-medium")}>
          {thread.title || t("Untitled conversation")}
        </span>
        <span className="text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs">
          {live ? (
            <LiveReplyLabel />
          ) : (
            <>
              <AgentTile agent={agent} size="xs" className="size-4" />
              <span className="truncate">{agent?.name ?? t("Agent unavailable")}</span>
            </>
          )}
        </span>
      </NavLink>
      <span className="absolute inset-y-0 right-1 flex items-center gap-0.5">
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
          aria-pressed={thread.pinned}
          className={cn(
            "transition-opacity focus-visible:opacity-100",
            thread.pinned
              ? "text-foreground"
              : "text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 group-focus-within:opacity-100",
          )}
          onClick={onTogglePin}
        >
          <PinIcon className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Delete conversation")}
          className="text-muted-foreground hover:text-destructive opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100"
          onClick={onDelete}
        >
          <Trash2Icon className="size-3.5" />
        </Button>
      </span>
    </div>
  );
}
