import { AgentPicker } from "@/components/assistant/agent-picker";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { RECENT_AGENT_LIMIT } from "@/components/assistant/use-askable-agent";
import { WorkingDot } from "@/components/assistant/voice/working-dot";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { agentRecency } from "@/lib/recent-agents";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
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
  EllipsisIcon,
  HomeIcon,
  InboxIcon,
  MessageSquareIcon,
  PanelLeftCloseIcon,
  PanelLeftOpenIcon,
  PinIcon,
  PlusIcon,
  RadarIcon,
  SearchIcon,
  Trash2Icon,
  TruckIcon,
  XIcon,
  type LucideIcon,
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
/** The keystroke that focuses the search. */
export const RAIL_SEARCH_SHORTCUT = "⌘K";

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

/** What the rail reads about the room: the counts on its places. */
function useRailCounts() {
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const { data: watchtowerCounts } = useQuery({
    ...queries.watchtower.counts(),
    enabled: canWatch,
  });

  return {
    canDecide,
    canWatch,
    decisions: attention?.agentDecisions ?? 0,
    unseen: watchtowerCounts?.unseen ?? 0,
    critical: watchtowerCounts?.unseenCritical ?? 0,
  };
}

/**
 * The Desk's left rail: where the Desk can go and every conversation there is.
 *
 * It reads top to bottom the way a day does — the way in first, then the
 * places, then the conversations shelved by when they were last touched,
 * what the person pinned above the calendar. Each row is a small card: the
 * title, and under it who it is with and when. A conversation being answered
 * says so in the agent's place.
 *
 * It folds to a strip of its places with ⌘B, and the fold is remembered.
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
  const counts = useRailCounts();
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
      <div className="flex h-11 shrink-0 items-center gap-1 pr-1.5 pl-3">
        <Link
          to="/desk"
          onClick={onNavigate}
          className="ui-focus-ring flex min-w-0 items-center gap-2 rounded-md py-1 pr-2"
        >
          <DeskMark />
          <span className="truncate text-sm font-semibold">{t("Desk")}</span>
        </Link>
        <span className="flex-1" />
        <RailAction label={collapse.label} shortcut={collapse.shortcut} onClick={onCollapse}>
          <PanelLeftCloseIcon className="size-4" />
        </RailAction>
      </div>

      <div className="flex shrink-0 flex-col gap-1.5 px-2 pt-1 pb-2">
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
            ) : (
              <Kbd className="text-2xs pointer-events-none">{RAIL_SEARCH_SHORTCUT}</Kbd>
            )
          }
        />
      </div>

      <nav aria-label={t("Desk")} className="flex shrink-0 flex-col gap-px px-2">
        <RailLink to="/desk" end icon={HomeIcon} label={t("Today")} onNavigate={onNavigate} />
        {counts.canWatch && (
          <RailLink
            to="/desk/watchtower"
            icon={RadarIcon}
            label={t("Watchtower")}
            onNavigate={onNavigate}
            count={counts.unseen}
            urgent={counts.critical > 0}
          />
        )}
        {counts.canDecide && (
          <RailLink
            to="/desk/decisions"
            icon={InboxIcon}
            label={t("Decisions")}
            onNavigate={onNavigate}
            count={counts.decisions}
          />
        )}
      </nav>

      <ScrollArea className="mt-2 min-h-0 flex-1" maskHeight={12}>
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

      <footer className="border-desk-hairline flex h-12 shrink-0 items-center gap-2 border-t px-2">
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
        <BackToTrenova />
      </footer>
    </div>
  );
}

/**
 * The rail folded to a strip: the way in, the places and the way out, as
 * marks with their names in tooltips. The conversations are behind the
 * unfold; a column of fifty untitled marks would say nothing.
 */
export function DeskRailStrip({
  isStarting,
  threads,
  onStart,
  onExpand,
}: {
  isStarting: boolean;
  threads: AssistantThread[];
  onStart: (agentId: string) => void;
  onExpand: () => void;
}) {
  const t = useT();
  const counts = useRailCounts();
  const user = useAuthStore((state) => state.user);

  return (
    <div
      data-slot="desk-rail-strip"
      className="bg-desk-rail flex h-full w-full flex-col items-center overflow-hidden"
    >
      <div className="flex h-11 shrink-0 items-center">
        <RailAction label={t("Show the rail")} shortcut={RAIL_SHORTCUT} onClick={onExpand}>
          <PanelLeftOpenIcon className="size-4" />
        </RailAction>
      </div>
      <div className="flex shrink-0 flex-col items-center gap-1 pt-1 pb-2">
        <NewConversationPicker threads={threads} disabled={isStarting} onStart={onStart} compact />
      </div>
      <nav aria-label={t("Desk")} className="flex shrink-0 flex-col items-center gap-1">
        <StripLink to="/desk" end icon={HomeIcon} label={t("Today")} />
        {counts.canWatch && (
          <StripLink
            to="/desk/watchtower"
            icon={RadarIcon}
            label={t("Watchtower")}
            count={counts.unseen}
            urgent={counts.critical > 0}
          />
        )}
        {counts.canDecide && (
          <StripLink
            to="/desk/decisions"
            icon={InboxIcon}
            label={t("Decisions")}
            count={counts.decisions}
          />
        )}
      </nav>
      <span className="flex-1" />
      <div className="flex h-12 shrink-0 flex-col items-center justify-center gap-1">
        <Tooltip>
          <TooltipTrigger
            render={
              <span className="flex">
                <ResolvedUserAvatar
                  userId={user?.id}
                  name={user?.name}
                  profilePicUrl={user?.profilePicUrl}
                  thumbnailUrl={user?.thumbnailUrl}
                  className="size-7"
                  fallbackClassName="bg-muted text-2xs font-medium text-muted-foreground"
                />
              </span>
            }
          />
          <TooltipContent side="right">{user?.name ?? t("Signed in")}</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}

function DeskMark() {
  return (
    <span
      aria-hidden
      className="bg-ink text-ink-foreground flex size-6 shrink-0 items-center justify-center rounded-md"
    >
      <TruckIcon className="size-3.5" />
    </span>
  );
}

function BackToTrenova() {
  const t = useT();

  return (
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
      <TooltipContent side="right" className="flex items-center gap-2">
        {label}
        {shortcut && <Kbd>{shortcut}</Kbd>}
      </TooltipContent>
    </Tooltip>
  );
}

/** A count on a place: quiet figures, in the danger tone only when something is critical. */
function RailCount({ count, urgent }: { count: number; urgent: boolean }) {
  if (count <= 0) {
    return null;
  }

  return (
    <span
      className={cn(
        "ml-auto shrink-0 text-xs tabular-nums",
        urgent ? "text-danger font-medium" : "text-foreground-subtle",
      )}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
}

function RailLink({
  to,
  end,
  icon: Icon,
  label,
  count = 0,
  urgent = false,
  onNavigate,
}: {
  to: string;
  end?: boolean;
  icon: LucideIcon;
  label: string;
  count?: number;
  urgent?: boolean;
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
          <RailCount count={count} urgent={urgent} />
        </>
      )}
    </NavLink>
  );
}

function StripLink({
  to,
  end,
  icon: Icon,
  label,
  count = 0,
  urgent = false,
}: {
  to: string;
  end?: boolean;
  icon: LucideIcon;
  label: string;
  count?: number;
  urgent?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <NavLink
            to={to}
            end={end}
            aria-label={label}
            className={({ isActive }) =>
              cn(
                "ui-focus-ring relative flex size-9 items-center justify-center rounded-md transition-colors",
                isActive
                  ? "bg-surface-selected text-foreground"
                  : "text-muted-foreground hover:bg-surface-hover hover:text-foreground",
              )
            }
          />
        }
      >
        <Icon className="size-4" />
        {count > 0 && (
          <span
            aria-hidden
            className={cn(
              "absolute top-1.5 right-1.5 size-1.5 rounded-full",
              urgent ? "bg-danger" : "bg-foreground-subtle",
            )}
          />
        )}
      </TooltipTrigger>
      <TooltipContent side="right" className="flex items-center gap-2">
        {label}
        {count > 0 && <span className="tabular-nums opacity-70">{count > 99 ? "99+" : count}</span>}
      </TooltipContent>
    </Tooltip>
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
  compact = false,
}: {
  threads: AssistantThread[];
  disabled: boolean;
  onStart: (agentId: string) => void;
  compact?: boolean;
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
        compact ? (
          <Button type="button" size="icon" aria-label={t("New conversation")}>
            <PlusIcon className="size-4" />
          </Button>
        ) : (
          <Button type="button" className="h-9 w-full justify-start px-3 has-[>svg]:px-3">
            <PlusIcon className="size-4" />
            <span className="min-w-0 flex-1 truncate text-left">{t("New conversation")}</span>
            <Kbd className="text-2xs bg-ink-foreground/15 text-ink-foreground/80 border-0">⌘N</Kbd>
          </Button>
        )
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
      {Array.from({ length: 6 }, (_, index) => (
        <Skeleton key={index} className="h-11" />
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
        <RailShelf
          key={shelf.key}
          shelf={shelf}
          start={starts[shelfIndex]}
          agentsById={agentsById}
          activeThreadId={activeThreadId}
          liveThreadIds={liveThreadIds}
          onDelete={onDelete}
          onTogglePin={onTogglePin}
          onNavigate={onNavigate}
        />
      ))}
    </div>
  );
}

function RailShelf({
  shelf,
  start,
  agentsById,
  activeThreadId,
  liveThreadIds,
  onDelete,
  onTogglePin,
  onNavigate,
}: {
  shelf: DeskThreadShelf;
  start: number;
  agentsById: ReadonlyMap<string, AgentChoice>;
  activeThreadId: string | null;
  liveThreadIds: ReadonlySet<string>;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
  onNavigate?: () => void;
}) {
  const t = useT();
  const heading = shelfHeading(t, shelf.key);

  return (
    <section aria-label={heading} className="flex flex-col gap-px">
      <h3 className="text-foreground-subtle flex h-7 items-center px-2.5 text-xs font-medium">
        {heading}
      </h3>
      {shelf.threads.map((thread, index) => (
        <RailRow
          key={thread.id}
          thread={thread}
          agent={agentsById.get(thread.agentDefinitionId) ?? null}
          active={thread.id === activeThreadId}
          live={liveThreadIds.has(thread.id)}
          style={{
            animationDelay: `${Math.min(start + index, ROW_STAGGER_CAP) * ROW_STAGGER_MS}ms`,
          }}
          onDelete={() => onDelete(thread)}
          onTogglePin={() => onTogglePin(thread)}
          onNavigate={onNavigate}
        />
      ))}
    </section>
  );
}

/**
 * One conversation: its title, on one line, and nothing else. Which agent
 * it is with is in the row's tooltip and the search, not on every row; a
 * reply being written is a dot before the title; the pin and the delete
 * are behind one "more" control that shows on hover. A list of fifty of
 * these should read as a list of titles, not a table.
 */
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
  const [menuOpen, setMenuOpen] = useState(false);
  const title = thread.title || t("Untitled conversation");

  return (
    <div
      data-thread-row
      data-state={menuOpen ? "open" : undefined}
      style={style}
      className={cn(
        "group animate-rise relative flex h-8 items-center rounded-md transition-colors",
        active
          ? "bg-surface-selected"
          : "hover:bg-surface-hover data-[state=open]:bg-surface-hover",
      )}
    >
      <NavLink
        to={conversationPath(thread.id)}
        onClick={onNavigate}
        title={agent ? t("With {0}", agent.name) : undefined}
        className={cn(
          "ui-inset-focus-ring flex h-full min-w-0 flex-1 items-center gap-2 rounded-md pr-8 pl-2.5 text-sm",
          active ? "text-foreground" : "text-foreground-muted group-hover:text-foreground",
        )}
      >
        {live && (
          <>
            <WorkingDot working className="shrink-0" />
            <span className="sr-only">{t("Replying")}</span>
          </>
        )}
        <span className="min-w-0 flex-1 truncate">{title}</span>
      </NavLink>
      <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={t("Conversation actions")}
              className={cn(
                "text-foreground-subtle hover:text-foreground absolute right-1 opacity-0 transition-opacity",
                "group-hover:opacity-100 group-focus-within:opacity-100 data-[popup-open]:opacity-100",
              )}
            />
          }
        >
          <EllipsisIcon className="size-3.5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" side="right" sideOffset={6} className="min-w-44">
          <DropdownMenuItem
            onClick={onTogglePin}
            startContent={<PinIcon className="size-4" />}
            title={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
          />
          <DropdownMenuItem
            color="danger"
            onClick={onDelete}
            startContent={<Trash2Icon className="size-4" />}
            title={t("Delete conversation")}
          />
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
