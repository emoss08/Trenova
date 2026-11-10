"use no memo";
import logo from "@/assets/logo.webp";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import {
  deskRailRows,
  type DeskRailRow,
  type DeskRailShelf,
} from "@/components/desk-chat/rail/desk-rail-rows";
import type { DeskShelfKey } from "@/components/desk-chat/rail/desk-threads";
import { shelfHeading } from "@/components/desk-chat/rail/rail-parts";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import type { AssistantThread } from "@/types/assistant";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { formatModShiftShortcut, formatShortcut } from "@trenova/shared/lib/shortcuts";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMemo, type ReactNode } from "react";
import { Link, useLocation, useSearchParams } from "react-router";
import { casePath } from "../attention/attention-record";
import { RailList } from "./rail-list";
import { RailRow, RailThreadRow } from "./rail-row";
import "./rail.css";

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace =
  | "today"
  | "needs"
  | "case"
  | "handled"
  | "triage"
  | "decisions"
  | "scheduled"
  | "memory"
  | "thread"
  | "agent";

/** A square icon button of the rail's top and foot. */
function RailIconButton({
  label,
  hint,
  bordered = false,
  onClick,
  children,
}: {
  label: string;
  hint: string;
  bordered?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      className={bordered ? "dk-rl-ib dk-bd ui-focus-ring" : "dk-rl-ib ui-focus-ring"}
      title={hint}
      aria-label={label}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

/** One of the Desk's places, or a lens indented under Needs you; a count only while something waits. */
function RailPlace({
  to,
  current,
  icon,
  label,
  count = 0,
}: {
  to: string;
  current: boolean;
  /** Left out for a lens. */
  icon?: DeskIconName;
  label: string;
  count?: number;
}) {
  return (
    <Link
      to={to}
      aria-current={current ? "page" : undefined}
      className={icon ? "dk-rl-r ui-focus-ring" : "dk-rl-r dk-sub ui-focus-ring"}
    >
      {icon && <DeskIcon name={icon} size={14} />}
      <span className="dk-rl-rt">{label}</span>
      {count > 0 && <span className="dk-rl-n">{count}</span>}
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
  /** Reads the next page of a shelf, or its first again after it failed. */
  onLoadMore: (shelf: DeskShelfKey) => void;
  /** Told which conversations the rail draws, so a page left stale is read when it shows. */
  onThreadsInView?: (threadIds: readonly string[]) => void;
  agentsById: ReadonlyMap<string, AgentChoice>;
  activeThreadId: string | null;
  canWatch: boolean;
  canDecide: boolean;
  canTriage: boolean;
  /** Open items in each lens of Needs you, from the Watchtower's counts. */
  lensCounts: { everything: number; watchtower: number; triage: number; decisions: number };
  onSearch: () => void;
  onSettings: () => void;
  /** The rail is folded away beside the page. */
  folded: boolean;
  /** Folds the rail away, or on a phone puts its sheet back. */
  onFold: () => void;
};

/**
 * The Desk's rail, as the approved design draws it: its fold, the logo (the
 * way back to Trenova), search and a new conversation across the top; Today,
 * Needs you with its three lenses, Scheduled and Memory; the cases opened
 * this session; every conversation under the day it was last touched; and
 * the person, with the keys and the Desk's settings, at the foot. Folded, it
 * takes no room and no focus; on a phone it is a sheet over the page.
 *
 * Only the rows near the viewport are mounted, and each shelf's next page is
 * read as the reader nears the end of what is loaded, so a rail of thousands
 * costs what a screenful does.
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
  onSearch,
  onSettings,
  folded,
  onFold,
}: DeskRailProps) {
  const t = useT();
  const rt = useRichT();
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
  const rows = useMemo(() => deskRailRows(shelves), [shelves]);
  const [search] = useSearchParams();
  const { pathname } = useLocation();
  const lens = place === "needs" ? (search.get("lens") ?? "") : "";
  const openWork = useDeskMarginStore((state) => state.openWork);
  const marginThreadId = useDeskMarginStore((state) => state.margin?.threadId ?? null);
  const inRoom = place === "thread";
  const openBeside = useMemo(
    () =>
      inRoom
        ? undefined
        : (thread: AssistantThread) => useDeskMarginStore.getState().showThread(thread.id, null),
    [inRoom],
  );
  const name = user?.name ?? user?.username ?? "";

  const renderRow = (row: DeskRailRow) => {
    switch (row.kind) {
      case "shelf":
        return <div className="dk-rl-gh">{shelfHeading(t, row.key)}</div>;
      case "thread":
        return (
          <RailThreadRow
            thread={row.thread}
            agentName={agentsById.get(row.thread.agentDefinitionId)?.name ?? null}
            live={liveThreadIds.has(row.thread.id)}
            current={inRoom ? row.thread.id === activeThreadId : row.thread.id === marginThreadId}
            today={today}
            timezone={timezone}
            onOpenBeside={openBeside}
          />
        );
      default:
        return row.failed ? (
          <div className="dk-rl-more dk-bad" role="alert">
            <span>{t("Couldn't load more conversations")}</span>
            <button type="button" className="dk-rl-retry" onClick={() => onLoadMore(row.shelf)}>
              {t("Try again")}
            </button>
          </div>
        ) : (
          <div
            className="dk-rl-more"
            role="status"
            aria-label={row.loading ? t("Loading more...") : undefined}
          >
            {row.loading && <span className="dk-sk dk-rl-skel" />}
          </div>
        );
    }
  };

  return (
    <aside
      className="dk-rl"
      aria-label={t("Desk")}
      data-folded={folded || undefined}
      inert={folded || undefined}
    >
      <div className="dk-rl-in">
        <div className="dk-rl-top">
          <RailIconButton
            label={t("Hide the rail")}
            hint={t("Hide the rail ({0})", formatShortcut("B"))}
            onClick={onFold}
          >
            <DeskIcon name="rail" size={14} />
          </RailIconButton>
          <Link to="/" className="dk-rl-logo ui-focus-ring" aria-label={t("Back to Trenova")}>
            <img src={logo} alt="" />
          </Link>
          <span className="dk-rl-sp" />
          <RailIconButton
            label={t("Search or run a command")}
            hint={t("Search ({0})", formatShortcut("K"))}
            onClick={onSearch}
          >
            <DeskIcon name="search" size={14} />
          </RailIconButton>
          <RailIconButton
            label={t("New conversation")}
            hint={t("New conversation ({0})", formatModShiftShortcut("O"))}
            bordered
            onClick={() => useDeskMarginStore.getState().openMargin(null)}
          >
            <DeskIcon name="plus" size={14} />
          </RailIconButton>
        </div>

        <RailList
          rows={rows}
          renderRow={renderRow}
          onLoadMore={onLoadMore}
          onThreadsInView={onThreadsInView}
          nav={
            <>
              <RailPlace to="/desk" current={place === "today"} icon="home" label={t("Today")} />
              {canWatch && (
                <RailPlace
                  to="/desk/needs"
                  current={place === "needs" && lens === ""}
                  icon="stack"
                  label={t("Needs you")}
                  count={lensCounts.everything}
                />
              )}
              {canWatch && (
                <RailPlace
                  to="/desk/needs?lens=watchtower"
                  current={lens === "watchtower"}
                  label={t("Watchtower")}
                  count={lensCounts.watchtower}
                />
              )}
              {canTriage && (
                <RailPlace
                  to="/desk/needs?lens=triage"
                  current={lens === "triage" || place === "triage"}
                  label={t("Triage")}
                  count={lensCounts.triage}
                />
              )}
              {canDecide && (
                <RailPlace
                  to="/desk/needs?lens=decisions"
                  current={lens === "decisions" || place === "decisions"}
                  label={t("Decisions")}
                  count={lensCounts.decisions}
                />
              )}
              <RailPlace
                to="/desk/scheduled"
                current={place === "scheduled"}
                icon="clock"
                label={t("Scheduled")}
              />
              <RailPlace
                to="/desk/memory"
                current={place === "memory"}
                icon="book"
                label={t("Memory")}
              />
              {openWork.length > 0 && (
                <>
                  <div className="dk-rl-gh">{t("Open work")}</div>
                  {openWork.map((work) => {
                    const to = casePath(work.kind, work.id);
                    return (
                      <RailRow
                        key={to}
                        to={to}
                        title={work.label}
                        sub={t("Case")}
                        current={pathname === to}
                      />
                    );
                  })}
                </>
              )}
            </>
          }
          empty={
            noConversations ? (
              <>
                <div className="dk-rl-gh">{t("Conversations")}</div>
                <p className="dk-rl-empty">
                  {rt("Your chats will show up here. Press <kbd/> to start one.", {
                    kbd: () => <span className="dk-kbd">{formatModShiftShortcut("O")}</span>,
                  })}
                </p>
              </>
            ) : null
          }
        />

        <div className="dk-rl-me">
          <span className="dk-rl-av">{getNameInitials(name, "")}</span>
          <b title={name}>{name}</b>
          <RailIconButton
            label={t("Keyboard shortcuts")}
            hint={t("Keyboard shortcuts (?)")}
            onClick={() => setDialogOpen("shortcuts", true)}
          >
            <DeskIcon name="keyboard" size={14} />
          </RailIconButton>
          <RailIconButton label={t("Desk settings")} hint={t("Settings")} onClick={onSettings}>
            <DeskIcon name="gear" size={14} />
          </RailIconButton>
        </div>
      </div>
    </aside>
  );
}
