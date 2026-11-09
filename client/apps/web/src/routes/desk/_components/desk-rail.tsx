"use no memo";
import logo from "@/assets/logo.webp";
import { DeleteConversationCard } from "@/components/assistant/delete-conversation-card";
import { useDeleteConversation } from "@/components/assistant/use-delete-conversation";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { Button } from "@trenova/shared/components/ui/button";
import { useMemo, useState } from "react";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { casePath } from "./attention/attention-record";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import {
  deskRailRows,
  type DeskRailRow,
  type DeskRailShelf,
} from "@/components/desk-chat/rail/desk-rail-rows";
import type { DeskShelfKey } from "@/components/desk-chat/rail/desk-threads";
import { shelfHeading } from "@/components/desk-chat/rail/rail-parts";
import { useRichT } from "@trenova/shared/i18n/rich";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { DeskRailFrame } from "./desk-rail-frame";
import { DeskRailList } from "./desk-rail-list";
import { DeskRailThreadRow, type DeskRailRowActions } from "./desk-rail-row";
import { formatModShiftShortcut, formatShortcut } from "@trenova/shared/lib/shortcuts";

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace =
  | "today"
  | "needs"
  | "case"
  | "triage"
  | "watchtower"
  | "decisions"
  | "scheduled"
  | "memory"
  | "thread"
  | "agent";

/** How long a deleted row takes to slide out before it leaves the list. */
const ROW_LEAVE_MS = 260;

const NO_IDS: ReadonlySet<string> = new Set();

const NAV_ITEM_CLASS =
  "relative z-1 flex h-7.5 w-full gap-2.5 rounded-lg pr-2 pl-2.5 text-left text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-fg/4 hover:text-dsk-fg";
const NAV_ITEM_ON_CLASS = "text-dsk-fg hover:bg-transparent";
const NAV_LABEL_CLASS = "min-w-0 flex-1 truncate";

/**
 * One of the Desk's places on the rail, as a link: it opens in a new tab like any
 * link, says it is the current page, and shows a count only when something waits.
 */
function RailPlace({
  to,
  placeKey,
  activeKey,
  icon,
  label,
  count = 0,
  waitsHere = false,
  sub = false,
}: {
  to: string;
  placeKey: string;
  activeKey: string | null;
  icon: DeskIconName;
  label: string;
  count?: number;
  /** A change waits on the person in the conversation they have open. */
  waitsHere?: boolean;
  /** A lens of the place above it, drawn under it without an icon. */
  sub?: boolean;
}) {
  const active = activeKey === placeKey;
  return (
    <Link
      to={to}
      data-k={placeKey}
      aria-current={active ? "page" : undefined}
      className={cn(
        "ui-focus-ring items-center outline-none",
        NAV_ITEM_CLASS,
        sub && "h-6.5 pl-8.5 text-xs",
        active && NAV_ITEM_ON_CLASS,
      )}
    >
      {!sub && <DeskIcon name={icon} size={14} />}
      <span className={NAV_LABEL_CLASS}>{label}</span>
      {count > 0 && <em className={cn("dk-sb-ct", waitsHere && "dk-w")}>{count}</em>}
    </Link>
  );
}

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
  canTriage: boolean;
  /** Open items in each lens of Needs you, from the Watchtower's counts. */
  lensCounts: { everything: number; watchtower: number; triage: number };
  decisionsCount: number;
  /** The open conversation has a change waiting on the person. */
  decisionsWaitHere: boolean;
  onSearch: () => void;
  onSettings: () => void;
  onSetPinned: (thread: AssistantThread, pinned: boolean) => void;
  onDelete: (thread: AssistantThread) => void;
  /** The rail is folded away beside the page. */
  folded: boolean;
  /** Folds the rail away, or on a phone puts its sheet back. */
  onFold: () => void;
};

/**
 * The Desk's rail: its fold, the logo, search and a new conversation across the top,
 * the Desk's places, then every conversation shelved by when it was last
 * touched. Each shelf folds away from its heading, and a folded shelf reads
 * nothing more. Each conversation is one line
 * with a dot that says what it is waiting on, and the current place sits on
 * a raised card that slides to whatever is chosen. Pin and delete come
 * forward on hover in place of the time; delete asks first, in a dialog, only
 * for a conversation with scheduled requests, which go with it.
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
  canTriage,
  lensCounts,
  decisionsCount,
  decisionsWaitHere,
  onSearch,
  onSettings,
  onSetPinned,
  onDelete,
  folded,
  onFold,
}: DeskRailProps) {
  const t = useT();
  const rt = useRichT();
  const navigate = useNavigate();
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
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

  const [search] = useSearchParams();
  const { pathname } = useLocation();
  const lensParam = place === "needs" ? (search.get("lens") ?? "") : "";
  const openWork = useDeskMarginStore((state) => state.openWork);
  const activeKey =
    place === "thread" && activeThreadId
      ? `c:${activeThreadId}`
      : place === "needs" && lensParam === "watchtower"
        ? "n:watch"
        : place === "needs" && lensParam === "triage"
          ? "n:triage"
          : place === "needs" && lensParam === "decisions"
            ? "n:dec"
            : place === "needs"
              ? "n:needs"
              : place === "case"
                ? `w:${pathname}`
                : place === "triage"
                  ? "n:triage"
                  : place === "decisions"
                    ? "n:dec"
              : place === "scheduled"
                ? "n:sched"
                : place === "memory"
                  ? "n:memory"
                  : place === "agent"
                    ? "n:agent"
                    : "n:today";

  // A conversation with schedules asks before it goes; any other slides out
  // at once.
  const deletion = useDeleteConversation((thread) => {
    setLeaving((current) => new Set(current).add(thread.id));
    window.setTimeout(() => onDelete(thread), ROW_LEAVE_MS);
  });
  const askDelete = useLatestCallback(deletion.ask);

  // One object for every row, kept while what it calls stays the same, so a
  // row re-renders only when its own conversation or state changes.
  const actions = useMemo<DeskRailRowActions>(
    () => ({
      setPinned: onSetPinned,
      askDelete,
    }),
    [askDelete, onSetPinned],
  );

  const renderRow = (row: DeskRailRow) => {
    switch (row.kind) {
      case "shelf": {
        return (
          <Button
            variant="bare"
            size="bare"
            className="dk-sb-gh dk-sb-gt"
            aria-expanded={!row.collapsed}
            onClick={() => toggleShelf(row.key)}
          >
            <span className="min-w-0 flex-1 truncate">{shelfHeading(t, row.key)}</span>
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
    <DeskRailFrame folded={folded} onFold={onFold}>
      <div className="dk-sb-top">
        <Button
          variant="quiet"
          size="icon-sm"
          className={deskIconClass}
          title={t("Hide sidebar")}
          aria-label={t("Hide sidebar")}
          onClick={onFold}
        >
          <DeskIcon name="rail" size={15} />
        </Button>
        <img className="dk-sb-logo" src={logo} alt={t("Trenova")} />
        <span style={{ flex: 1 }} />
        <Button
          variant="quiet"
          size="icon-sm"
          className={deskIconClass}
          title={t("Search ({0})", formatShortcut("K"))}
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
          title={t("New conversation ({0})", formatModShiftShortcut("O"))}
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
            <RailPlace
              to="/desk"
              placeKey="n:today"
              activeKey={activeKey}
              icon="home"
              label={t("Today")}
            />
            {canWatch && (
              <RailPlace
                to="/desk/needs"
                placeKey="n:needs"
                activeKey={activeKey}
                icon="stack"
                label={t("Needs you")}
                count={lensCounts.everything}
              />
            )}
            {canWatch && (
              <RailPlace
                to="/desk/needs?lens=watchtower"
                placeKey="n:watch"
                activeKey={activeKey}
                icon="radar"
                label={t("Watchtower")}
                count={lensCounts.watchtower}
                sub
              />
            )}
            {canTriage && (
              <RailPlace
                to="/desk/needs?lens=triage"
                placeKey="n:triage"
                activeKey={activeKey}
                icon="mail"
                label={t("Triage")}
                count={lensCounts.triage}
                sub
              />
            )}
            {canDecide && (
              <RailPlace
                to="/desk/needs?lens=decisions"
                placeKey="n:dec"
                activeKey={activeKey}
                icon="inbox"
                label={t("Decisions")}
                count={decisionsCount}
                waitsHere={decisionsWaitHere}
                sub
              />
            )}
            <RailPlace
              to="/desk/scheduled"
              placeKey="n:sched"
              activeKey={activeKey}
              icon="clock"
              label={t("Scheduled")}
            />
            <RailPlace
              to="/desk/memory"
              placeKey="n:memory"
              activeKey={activeKey}
              icon="memory"
              label={t("Memory")}
            />
            {openWork.length > 0 && (
              <>
                <div className="dk-sb-gh">{t("Open work")}</div>
                {openWork.map((work) => {
                  const to = casePath(work.kind, work.id);
                  const active = activeKey === `w:${to}`;
                  return (
                    <Link
                      key={to}
                      to={to}
                      aria-current={active ? "page" : undefined}
                      className={cn("dk-w-sbw ui-focus-ring", active && "dk-on")}
                    >
                      <span className="dk-w-sbw-d" />
                      <span className="dk-w-sbw-tx">
                        <span className="dk-w-sbw-t">{work.label}</span>
                        <span className="dk-w-sbw-s">{t("Case")}</span>
                      </span>
                    </Link>
                  );
                })}
              </>
            )}
          </>
        }
        empty={
          noConversations ? (
            <>
              <div className="dk-sb-gh">{t("Conversations")}</div>
              <div className="dk-sb-empty">
                {rt("Your chats will show up here. Press <kbd/> to start one.", {
                  kbd: () => <span className="dk-kbd">{formatModShiftShortcut("O")}</span>,
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
          title={t("Keyboard shortcuts (?)")}
          aria-label={t("Keyboard shortcuts")}
          onClick={() => setDialogOpen("shortcuts", true)}
        >
          <DeskIcon name="keyboard" size={15} />
        </Button>
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
      <DeleteConversationCard
        thread={deletion.asking}
        onConfirm={deletion.confirm}
        onClose={deletion.cancel}
      />
    </DeskRailFrame>
  );
}
