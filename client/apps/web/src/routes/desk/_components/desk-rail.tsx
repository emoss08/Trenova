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
import { Button } from "@trenova/shared/components/ui/button";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { nextWake, useSnoozeClock } from "./case/use-snooze-clock";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskRailRows, type DeskRailRow } from "@/components/desk-chat/rail/desk-rail-rows";
import { groupDeskThreadsByRecency } from "@/components/desk-chat/rail/desk-threads";
import { shelfHeading } from "@/components/desk-chat/rail/rail-parts";
import { useRichT } from "@trenova/shared/i18n/rich";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { DeskRailList, type DeskRailPaging } from "./desk-rail-list";
import { DeskRailThreadRow, type DeskRailRowActions } from "./desk-rail-row";

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace = "today" | "watchtower" | "decisions" | "memory" | "thread" | "agent";

/** How long a deleted row takes to slide out before it leaves the list. */
const ROW_LEAVE_MS = 260;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

const NAV_ITEM_CLASS =
  "relative z-1 flex h-7.5 w-full gap-2.5 rounded-lg pr-2 pl-2.5 text-left text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-fg/4 hover:text-dsk-fg";
const NAV_ITEM_ON_CLASS = "text-dsk-fg hover:bg-transparent";
const NAV_LABEL_CLASS = "min-w-0 flex-1 truncate";

/** A rail with every conversation already read. */
const ALL_READ: DeskRailPaging = {
  hasMore: false,
  loadingMore: false,
  failed: false,
  loadMore: () => undefined,
};

export type DeskRailProps = {
  place: DeskPlace;
  /** The conversations read so far, pinned first and then newest first. */
  threads: readonly AssistantThread[];
  /** Whether more conversations remain to be read, and how to read them. */
  paging?: DeskRailPaging;
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
  onTogglePin: (thread: AssistantThread) => void;
  onDelete: (thread: AssistantThread) => void;
  /** A snoozed case's time came: read the conversations again for its state. */
  onCaseWake?: () => void;
};

/**
 * The Desk's rail: the logo, search and a new conversation across the top,
 * the three places, then every conversation shelved by when it was last
 * touched. Each conversation is one line with a dot that says what it is
 * waiting on, and the current place sits on a raised card that slides to
 * whatever is chosen. Pin and delete come forward on hover in place of the
 * time; delete asks once more, inline, before it acts.
 *
 * Only the rows near the viewport are mounted, and the next page of
 * conversations is read as the reader nears the end of what is loaded, so a
 * rail of thousands costs what a screenful does.
 */
export function DeskRail({
  place,
  threads,
  paging = ALL_READ,
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
  onTogglePin,
  onDelete,
  onCaseWake,
}: DeskRailProps) {
  const t = useT();
  const rt = useRichT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
  const [mountedAt] = useState(nowInSeconds);
  const wakeAt = useMemo(
    () =>
      nextWake(
        threads.map((thread) =>
          thread.case?.state === "Snoozed" ? thread.case.snoozedUntil : null,
        ),
        mountedAt,
      ),
    [mountedAt, threads],
  );
  const now = useSnoozeClock(wakeAt, onCaseWake);
  const [confirming, setConfirming] = useState<string | null>(null);
  const [leaving, setLeaving] = useState<string | null>(null);

  const shelves = useMemo(
    () => groupDeskThreadsByRecency(threads, now, { pinnedFirst: true, timezone }),
    [now, threads, timezone],
  );
  const rows = useMemo(() => deskRailRows(shelves, paging.hasMore), [paging.hasMore, shelves]);

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
      togglePin: onTogglePin,
      confirmDelete: setConfirming,
      cancelDelete: () => setConfirming(null),
      remove: (thread) => {
        setConfirming(null);
        setLeaving(thread.id);
        window.setTimeout(() => {
          setLeaving(null);
          onDelete(thread);
        }, ROW_LEAVE_MS);
      },
    }),
    [navigate, onDelete, onTogglePin],
  );

  const renderRow = (row: DeskRailRow) => {
    switch (row.kind) {
      case "shelf":
        return <div className="dk-sb-gh">{shelfHeading(t, row.key)}</div>;
      case "thread": {
        const { thread } = row;
        return (
          <DeskRailThreadRow
            thread={thread}
            agentName={agentsById.get(thread.agentDefinitionId)?.name ?? null}
            live={liveThreadIds.has(thread.id)}
            active={thread.id === activeThreadId && place === "thread"}
            confirming={confirming === thread.id}
            leaving={leaving === thread.id}
            now={now}
            timezone={timezone}
            actions={actions}
          />
        );
      }
      default:
        return paging.failed ? (
          <div className="dk-sb-more dk-sb-more-err" role="alert">
            <span>{t("Couldn't load more conversations")}</span>
            <Button variant="bare" size="bare" className="dk-sb-retry" onClick={paging.loadMore}>
              {t("Try again")}
            </Button>
          </div>
        ) : (
          <div
            className="dk-sb-more"
            role="status"
            aria-label={paging.loadingMore ? t("Loading more...") : undefined}
          >
            {paging.loadingMore && <span className="dk-sk dk-sb-skel" />}
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
        paging={paging}
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
          <>
            <div className="dk-sb-gh">{t("Conversations")}</div>
            <div className="dk-sb-empty">
              {rt("Your chats will show up here. Press <kbd/> to start one.", {
                kbd: () => <span className="dk-kbd">⌘N</span>,
              })}
            </div>
          </>
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
