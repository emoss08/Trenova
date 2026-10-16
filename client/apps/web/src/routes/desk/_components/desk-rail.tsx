"use no memo";
import logo from "@/assets/logo.webp";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { Button } from "@trenova/shared/components/ui/button";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import {
  deskRailRows,
  type DeskRailRow,
  type DeskRailShelf,
} from "@/components/desk-chat/rail/desk-rail-rows";
import type { DeskShelfKey } from "@/components/desk-chat/rail/desk-threads";
import { shelfHeading } from "@/components/desk-chat/rail/rail-parts";
import { useRichT } from "@trenova/shared/i18n/rich";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { DeskRailList } from "./desk-rail-list";
import { DeskRailThreadRow, type DeskRailRowActions } from "./desk-rail-row";

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace = "today" | "watchtower" | "decisions" | "memory" | "thread" | "agent";

/** How long a deleted row takes to slide out before it leaves the list. */
const ROW_LEAVE_MS = 260;

const NO_IDS: ReadonlySet<string> = new Set();

const NAV_ITEM_CLASS =
  "relative z-1 flex h-7.5 w-full gap-2.5 rounded-lg pr-2 pl-2.5 text-left text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-fg/4 hover:text-dsk-fg";
const NAV_ITEM_ON_CLASS = "text-dsk-fg hover:bg-transparent";
const NAV_LABEL_CLASS = "min-w-0 flex-1 truncate";

export type DeskRailProps = {
  place: DeskPlace;
  /** Every shelf in the rail's order, with what has been read of each. */
  shelves: readonly DeskRailShelf[];
  /** When the reader's day began, in Unix seconds: what each row's time is read against. */
  today: number;
  /** Every shelf was counted and none holds anything. */
  noConversations: boolean;
  /** Reads the next page of an open shelf, or its first again after it failed. */
  onLoadMore: (shelf: DeskShelfKey) => void;
  /** Told which conversations the rail draws, so a page left stale is read when it shows. */
  onThreadsInView?: (threadIds: readonly string[]) => void;
  agentsById: ReadonlyMap<string, AgentChoice>;
  activeThreadId: string | null;
  canWatch: boolean;
  canDecide: boolean;
  watchtowerCount: number;
  decisionsCount: number;
  /** The open conversation has a change waiting on the person. */
  decisionsWaitHere: boolean;
  onSearch: () => void;
  onSettings: () => void;
  onSetPinned: (thread: AssistantThread, pinned: boolean) => void;
  onDelete: (thread: AssistantThread) => void;
};

/**
 * The Desk's rail: the logo, search and a new conversation across the top,
 * the three places, then every conversation shelved by when it was last
 * touched. Each shelf folds away from its heading, and a folded shelf shows
 * how many it holds and reads nothing more. Each conversation is one line
 * with a dot that says what it is waiting on, and the current place sits on
 * a raised card that slides to whatever is chosen. Pin and delete come
 * forward on hover in place of the time; delete asks once more, inline,
 * before it acts.
 *
 * Only the rows near the viewport are mounted, and each open shelf's next
 * page is read as the reader nears the end of what is loaded, so a rail of
 * thousands costs what a screenful does.
 */
export function DeskRail({
  place,
  shelves,
  today,
  noConversations,
  onLoadMore,
  onThreadsInView,
  agentsById,
  activeThreadId,
  canWatch,
  canDecide,
  watchtowerCount,
  decisionsCount,
  decisionsWaitHere,
  onSearch,
  onSettings,
  onSetPinned,
  onDelete,
}: DeskRailProps) {
  const t = useT();
  const rt = useRichT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
  const [confirming, setConfirming] = useState<string | null>(null);
  const [leaving, setLeaving] = useState<ReadonlySet<string>>(NO_IDS);

  const toggleShelf = useDeskSettingsStore((state) => state.toggleShelf);
  const openThreadId = place === "thread" ? activeThreadId : null;
  const rows = useMemo(() => deskRailRows(shelves, openThreadId), [openThreadId, shelves]);

  // A deleted row stays slid out until the rail is given shelves without it.
  // The cache drops it as the delete is sent, but the rail hears of that a
  // task later and would draw the row again in between. Once it has left, a
  // refused delete that puts it back shows it as it was.
  if (leaving.size > 0) {
    const listed = new Set<string>();
    for (const row of rows) {
      if (row.kind === "thread" && leaving.has(row.thread.id)) {
        listed.add(row.thread.id);
      }
    }
    if (listed.size !== leaving.size) {
      setLeaving(listed.size === 0 ? NO_IDS : listed);
    }
  }

  const activeKey =
    place === "thread" && activeThreadId
      ? `c:${activeThreadId}`
      : place === "watchtower"
        ? "n:watch"
        : place === "decisions"
          ? "n:dec"
          : place === "memory"
            ? "n:memory"
            : place === "agent"
              ? "n:agent"
              : "n:today";

  // One object for every row, kept while what it calls stays the same, so a
  // row re-renders only when its own conversation or state changes.
  const actions = useMemo<DeskRailRowActions>(
    () => ({
      open: (thread) => void navigate(conversationPath(thread.id)),
      setPinned: onSetPinned,
      confirmDelete: setConfirming,
      cancelDelete: () => setConfirming(null),
      remove: (thread) => {
        setConfirming(null);
        setLeaving((current) => new Set(current).add(thread.id));
        window.setTimeout(() => onDelete(thread), ROW_LEAVE_MS);
      },
    }),
    [navigate, onDelete, onSetPinned],
  );

  const renderRow = (row: DeskRailRow) => {
    switch (row.kind) {
      case "shelf": {
        const heading = shelfHeading(t, row.key);
        const counted = row.collapsed && typeof row.count === "number" ? row.count : null;
        return (
          <Button
            variant="bare"
            size="bare"
            className="dk-sb-gh dk-sb-gt"
            aria-expanded={!row.collapsed}
            aria-label={
              counted === null
                ? heading
                : t("{0}, {1, plural, one {# conversation} other {# conversations}}", heading, counted)
            }
            onClick={() => toggleShelf(row.key)}
          >
            <span className="min-w-0 flex-1 truncate">{heading}</span>
            {counted !== null && (
              <em className="dk-sb-ct" aria-hidden>
                {counted}
              </em>
            )}
            <DeskIcon name="chevR" size={12} stroke={2} className="dk-sb-gc" />
          </Button>
        );
      }
      case "thread": {
        const { thread } = row;
        return (
          <DeskRailThreadRow
            thread={thread}
            agentName={agentsById.get(thread.agentDefinitionId)?.name ?? null}
            live={liveThreadIds.has(thread.id)}
            active={thread.id === activeThreadId && place === "thread"}
            confirming={confirming === thread.id}
            leaving={leaving.has(thread.id)}
            today={today}
            timezone={timezone}
            actions={actions}
          />
        );
      }
      default:
        return row.failed ? (
          <div className="dk-sb-more dk-sb-more-err" role="alert">
            <span>{t("Couldn't load more conversations")}</span>
            <Button
              variant="bare"
              size="bare"
              className="dk-sb-retry"
              onClick={() => onLoadMore(row.shelf)}
            >
              {t("Try again")}
            </Button>
          </div>
        ) : (
          <div
            className="dk-sb-more"
            role="status"
            aria-label={row.loading ? t("Loading more...") : undefined}
          >
            {row.loading && <span className="dk-sk dk-sb-skel" />}
          </div>
        );
    }
  };

  const name = user?.name ?? user?.username ?? "";

  return (
    <aside className="dk-sb" aria-label={t("Desk")}>
      <div className="dk-sb-top">
        <img className="dk-sb-logo" src={logo} alt={t("Trenova")} />
        <span style={{ flex: 1 }} />
        <Button
          variant="quiet"
          size="icon-sm"
          className={deskIconClass}
          title={t("Search  ⌘K")}
          aria-label={t("Search")}
          onClick={onSearch}
        >
          <DeskIcon name="search" size={15} />
        </Button>
        <Button
          variant="quiet"
          size="icon-sm"
          className={cn(
            deskIconClass,
            "dk-sb-new bg-dsk-card text-dsk-fg hover:text-dsk-fg [&_svg]:transition-transform [&_svg]:duration-[360ms] [&_svg]:ease-(--dk-spring) hover:[&_svg]:rotate-90",
          )}
          title={t("New conversation  ⌘N")}
          aria-label={t("New conversation")}
          onClick={() => void navigate("/desk")}
        >
          <DeskIcon name="plus" size={15} stroke={2} />
        </Button>
      </div>

      <DeskRailList
        activeKey={activeKey}
        rows={rows}
        renderRow={renderRow}
        onLoadMore={onLoadMore}
        onThreadsInView={onThreadsInView}
        nav={
          <>
            <Button
              variant="bare"
              size="bare"
              data-k="n:today"
              className={cn(NAV_ITEM_CLASS, activeKey === "n:today" && NAV_ITEM_ON_CLASS)}
              onClick={() => void navigate("/desk")}
            >
              <DeskIcon name="home" size={14} />
              <span className={NAV_LABEL_CLASS}>{t("Today")}</span>
            </Button>
            {canWatch && (
              <Button
                variant="bare"
                size="bare"
                data-k="n:watch"
                className={cn(NAV_ITEM_CLASS, activeKey === "n:watch" && NAV_ITEM_ON_CLASS)}
                onClick={() => void navigate("/desk/watchtower")}
              >
                <DeskIcon name="radar" size={14} />
                <span className={NAV_LABEL_CLASS}>{t("Watchtower")}</span>
                <em className="dk-sb-ct">{watchtowerCount}</em>
              </Button>
            )}
            {canDecide && (
              <Button
                variant="bare"
                size="bare"
                data-k="n:dec"
                className={cn(NAV_ITEM_CLASS, activeKey === "n:dec" && NAV_ITEM_ON_CLASS)}
                onClick={() => void navigate("/desk/decisions")}
              >
                <DeskIcon name="inbox" size={14} />
                <span className={NAV_LABEL_CLASS}>{t("Decisions")}</span>
                <em className={cn("dk-sb-ct", decisionsWaitHere && "dk-w")}>{decisionsCount}</em>
              </Button>
            )}
            <Button
              variant="bare"
              size="bare"
              data-k="n:memory"
              className={cn(NAV_ITEM_CLASS, activeKey === "n:memory" && NAV_ITEM_ON_CLASS)}
              onClick={() => void navigate("/desk/memory")}
            >
              <DeskIcon name="memory" size={14} />
              <span className={NAV_LABEL_CLASS}>{t("Memory")}</span>
            </Button>
          </>
        }
        empty={
          noConversations ? (
            <>
              <div className="dk-sb-gh">{t("Conversations")}</div>
              <div className="dk-sb-empty">
                {rt("Your chats will show up here. Press <kbd/> to start one.", {
                  kbd: () => <span className="dk-kbd">⌘N</span>,
                })}
              </div>
            </>
          ) : null
        }
      />

      <div className="dk-sb-me">
        <span className="dk-me">{getNameInitials(name, "")}</span>
        <b className="truncate w-[100px]" title={name}>
          {name}
        </b>
        <span style={{ flex: 1 }} />
        <Button
          variant="quiet"
          size="icon-sm"
          className={deskIconClass}
          title={t("Settings")}
          aria-label={t("Desk settings")}
          onClick={onSettings}
        >
          <DeskIcon name="gear" size={15} />
        </Button>
        <Button
          variant="quiet"
          size="icon-sm"
          className={cn(deskIconClass, "dk-sb-back relative overflow-hidden")}
          title={t("Back to Trenova")}
          aria-label={t("Back to Trenova")}
          onClick={() => void navigate("/")}
        >
          <span className="dk-sb-bk-c">
            <DeskIcon name="chevL" size={14} />
          </span>
          <img className="dk-sb-bk-l" src={logo} alt="" />
        </Button>
      </div>
    </aside>
  );
}
