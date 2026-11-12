"use no memo";
import logo from "@/assets/logo.webp";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import {
  deskRailRows,
  type DeskRailRow,
  type DeskRailShelf,
} from "@/components/desk-chat/rail/desk-rail-rows";
import type { DeskShelfEdges, DeskShelfKey } from "@/components/desk-chat/rail/desk-threads";
import { shelfHeading } from "@/components/desk-chat/rail/rail-parts";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useDeskRailSlotStore } from "@/stores/desk-rail-slot-store";
import { useTriageStore } from "@/stores/triage-store";
import { HoverGlideGroup } from "@/components/hover-glide";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import type { AssistantThread } from "@/types/assistant";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { formatModShiftShortcut, formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMemo, useState, type PointerEvent, type ReactNode } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router";
import { DeleteConversationCard } from "@/components/assistant/delete-conversation-card";
import { useDeleteConversation } from "@/components/assistant/use-delete-conversation";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { casePath } from "../attention/attention-record";
import { caseOfThread } from "../record-case/case-subject";
import { railCaseRows, type RailCaseRow } from "./open-work";
import { threadMenu, workMenu } from "./rail-menus";
import { RenameConversationDialog } from "./rename-conversation-dialog";
import { RailList } from "./rail-list";
import { RAIL_TRANSITION_CLASS } from "../desk-motion";
import { RailCaseRowItem, RailThreadRow } from "./rail-row";
import {
  RAIL_LIST_CLASS,
  RAIL_TOP_CLASS,
  railClass,
  railInnerClass,
  type RailMode,
} from "./rail-classes";

const NO_WAITING_CASES: readonly DeskAttentionGroup[] = [];

/** The rail is one of three, and the place in front decides which. */
export type RailView = "cases" | "chats" | "triage";

/**
 * Which rail a place belongs to. Needs you, a record's case and what Nova
 * handled are case work: their rail holds Needs you with its lenses and the
 * cases by day, and a way back. Everywhere else is the conversations' rail:
 * Today, Scheduled, Memory, the way into cases, and the chats by day. Triage
 * is a room with places of its own, its folders, and they take the rail while
 * it is in front, so there is one rail beside the page and never two.
 *
 * A conversation asked on a record's case is case work too. Opened on its own
 * it keeps the cases' rail, where its case is a row; in the conversations'
 * rail it would be a page with no row to be found on.
 */
export function railViewOf(place: DeskPlace, onACase = false): RailView {
  if (place === "thread" && onACase) {
    return "cases";
  }
  switch (place) {
    case "needs":
    case "case":
    case "handled":
    case "decisions":
      return "cases";
    case "triage":
      return "triage";
    default:
      return "chats";
  }
}

/** Where in the Desk a person was before Triage: the view, and the page's own address. */
export type TriageBack = { view: Exclude<RailView, "triage">; to: string };

/** Triage opened on its own (a link, a reload) leads back to the Desk's home. */
export const TRIAGE_BACK_HOME: TriageBack = { view: "chats", to: "/desk" };

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace =
  | "today"
  | "needs"
  | "case"
  | "handled"
  | "view"
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
      className={cn(
        "grid size-8 place-items-center rounded-[8px] ui-focus-ring hover:bg-dsk-hover hover:text-dsk-fg",
        bordered ? "border border-dsk-b text-dsk-fg" : "text-dsk-subtle",
      )}
      title={hint}
      aria-label={label}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

// i18n-ignore: class names, never shown to a person
const RAIL_MORE_CLASS = "flex min-h-7.5 items-center gap-2 px-2.5 text-dsk-sm text-dsk-subtle";

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
      viewTransition
      data-glide-target
      aria-current={current ? "page" : undefined}
      className={cn(
        "relative flex w-full items-center gap-2.5 rounded-[8px] ui-focus-ring text-left",
        "hover:text-dsk-fg aria-[current=page]:bg-dsk-canvas aria-[current=page]:text-dsk-fg",
        "aria-[current=page]:shadow-[0_0_0_1px_var(--dsk-b)] [&_svg]:text-dsk-subtle",
        icon
          ? "h-7.5 px-2.5 text-dsk-base text-dsk-fg2"
          : "h-6.5 pr-2.5 pl-8.5 text-sm text-dsk-subtle",
      )}
    >
      {icon && <DeskIcon name={icon} size={14} />}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {count > 0 && (
        <span
          data-slot="rail-count"
          className={cn(
            "text-xs leading-[inherit] tabular-nums",
            icon ? "text-dsk-subtle" : "text-dsk-faint",
          )}
        >
          {count}
        </span>
      )}
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
  lensCounts: { everything: number; watchtower: number; decisions: number };
  /** Messages waiting on a person in Triage, the number its own Waiting on you shows. */
  triageWaiting: number;
  /** Where Triage was opened from, which its way back leads to. */
  triageBack?: TriageBack;
  /** The records a plan is waiting on a person for, as Needs you orders them. */
  waitingCases?: readonly DeskAttentionGroup[];
  /** Which rail is drawn, from the place in front: the cases', or the conversations'. */
  view: RailView;
  /** The instants each shelf begins at, which shelve a case that has no conversation yet. */
  edges: DeskShelfEdges;
  /** What a conversation's row can do from its right-click menu. */
  onSetPinned: (thread: AssistantThread, pinned: boolean) => void;
  onRename: (thread: AssistantThread, title: string) => void;
  onDownload: (thread: AssistantThread) => void;
  onDelete: (thread: AssistantThread) => void;
  onSearch: () => void;
  onSettings: () => void;
  /** The rail is folded away beside the page. */
  folded: boolean;
  /** Pinned beside the page, or floating over it. */
  mode: RailMode;
  /** The rail is showing over the page: its sheet on a phone, or its floating card. */
  open: boolean;
  /** The pointer came onto the rail, or left it: a floating rail stays while it is on it. */
  onPointerEnter?: (event: PointerEvent<HTMLElement>) => void;
  onPointerLeave?: (event: PointerEvent<HTMLElement>) => void;
  /** Folds the rail away, or on a phone puts its sheet back. */
  onFold: () => void;
};

/**
 * The Desk's rail, as the approved design draws it: its fold, the logo (the
 * way back to Trenova), search and a new conversation across the top; Today,
 * Needs you with its three lenses, Scheduled and Memory; the records a plan
 * waits on and the cases opened this session; every conversation under the day it was last touched; and
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
  triageWaiting,
  triageBack = TRIAGE_BACK_HOME,
  waitingCases = NO_WAITING_CASES,
  view,
  edges,
  onSetPinned,
  onRename,
  onDownload,
  onDelete,
  onSearch,
  onSettings,
  folded,
  mode,
  open,
  onPointerEnter,
  onPointerLeave,
  onFold,
}: DeskRailProps) {
  const t = useT();
  const rt = useRichT();
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
  const [search] = useSearchParams();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const lens = place === "needs" ? (search.get("lens") ?? "") : "";
  const openWork = useDeskMarginStore((state) => state.openWork);
  const rows = useMemo(
    () =>
      view === "cases"
        ? railCaseRows(shelves, waitingCases, openWork, edges, today)
        : deskRailRows(shelves),
    [view, shelves, waitingCases, openWork, edges, today],
  );
  const marginThreadId = useDeskMarginStore((state) => state.margin?.threadId ?? null);
  const inRoom = place === "thread";
  // A click on a conversation always opens its own page. Beside the page is
  // offered from the row's menu, except where it could not sit: a conversation
  // already fills the room, and beside a case sits the case's own.
  const fillsTheMargin = inRoom || place === "case";
  const openBeside = useMemo(
    () =>
      fillsTheMargin
        ? undefined
        : (thread: AssistantThread) => useDeskMarginStore.getState().showThread(thread.id, null),
    [fillsTheMargin],
  );
  const name = user?.name ?? user?.username ?? "";
  const setSlot = useDeskRailSlotStore((state) => state.setSlot);

  // Every row of work has a menu on right-click or a long press: what can be
  // done to it without opening it first.
  const { copy } = useCopyToClipboard();
  const deletion = useDeleteConversation(onDelete);
  const [renaming, setRenaming] = useState<AssistantThread | null>(null);
  const forgetOpenWork = useDeskMarginStore((state) => state.forgetOpenWork);
  const openPath = (path: string) => void navigate(path);
  const openInNewTab = (path: string) => window.open(path, "_blank", "noopener,noreferrer");
  const copyLink = (path: string) =>
    void copy(new URL(path, window.location.origin).toString(), { withToast: true });
  const menuOfThread = (thread: AssistantThread) =>
    threadMenu(t, thread, {
      open: openPath,
      openBeside,
      openInNewTab,
      copyLink,
      setPinned: onSetPinned,
      rename: setRenaming,
      download: onDownload,
      remove: deletion.ask,
    });
  const menuOfWork = (work: RailCaseRow) =>
    work.thread
      ? menuOfThread(work.thread)
      : workMenu(t, work, {
          open: openPath,
          openInNewTab,
          copyLink,
          recordPath: (row) => {
            const kind = row.view ? null : caseKindOf(row.to);
            return kind !== null && isRecordEntityType(kind)
              ? recordPath(kind, row.recordId)
              : null;
          },
          forget: (row) => {
            const entry = openWork.find(
              (work) => (work.path ?? casePath(work.kind, work.id)) === row.to,
            );
            if (entry) {
              forgetOpenWork(entry.kind, entry.id);
            }
          },
        });

  const renderRow = (row: DeskRailRow) => {
    switch (row.kind) {
      case "shelf":
        return (
          <div
            data-slot="rail-heading"
            className="flex justify-between px-2.5 pt-4.5 pb-2 text-xs leading-none text-dsk-faint"
          >
            {shelfHeading(t, row.key)}
          </div>
        );
      case "case":
        return (
          <RailCaseRowItem
            row={row.case}
            agentName={
              row.case.thread
                ? (agentsById.get(row.case.thread.agentDefinitionId)?.name ?? null)
                : null
            }
            live={row.case.thread ? liveThreadIds.has(row.case.thread.id) : false}
            current={
              pathname === row.case.to ||
              (inRoom && row.case.thread !== null && row.case.thread.id === activeThreadId)
            }
            menu={() => menuOfWork(row.case)}
          />
        );
      case "thread":
        return (
          <RailThreadRow
            thread={row.thread}
            agentName={agentsById.get(row.thread.agentDefinitionId)?.name ?? null}
            live={liveThreadIds.has(row.thread.id)}
            current={
              inRoom
                ? row.thread.id === activeThreadId
                : row.thread.id === marginThreadId || pathname === caseRowPath(row.thread)
            }
            today={today}
            timezone={timezone}
            menu={menuOfThread}
          />
        );
      default:
        return row.failed ? (
          <div className={RAIL_MORE_CLASS} role="alert">
            <span className="flex-1 text-danger-subtle-foreground">
              {t("Couldn't load more conversations")}
            </span>
            <button
              type="button"
              className="rounded-[4px] font-medium text-dsk-fg hover:underline"
              onClick={() => onLoadMore(row.shelf)}
            >
              {t("Try again")}
            </button>
          </div>
        ) : (
          <div
            className={RAIL_MORE_CLASS}
            role="status"
            aria-label={row.loading ? t("Loading more...") : undefined}
          >
            {row.loading && <span className="dk-sk block h-2.5 w-[70%]" />}
          </div>
        );
    }
  };

  return (
    <aside
      className={railClass(mode)}
      aria-label={t("Desk")}
      data-mode={mode}
      data-folded={folded || undefined}
      data-open={open || undefined}
      onPointerEnter={onPointerEnter}
      onPointerLeave={onPointerLeave}
      inert={folded || (mode === "floating" && !open) || undefined}
    >
      <div className={railInnerClass(mode)}>
        <div className={RAIL_TOP_CLASS}>
          <RailIconButton
            label={t("Hide the rail")}
            hint={t("Hide the rail ({0})", formatShortcut("B"))}
            onClick={onFold}
          >
            <DeskIcon name="rail" size={14} />
          </RailIconButton>
          <Link
            to="/"
            className="grid place-items-center rounded-[6px] ui-focus-ring"
            aria-label={t("Back to Trenova")}
          >
            <img src={logo} alt="" className="size-6.5" />
          </Link>
          <span className="flex-1" />
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
            onClick={() => {
              // Beside a case sits the case's conversation, so a new one
              // starts on the Desk's own page.
              if (place === "case") {
                void navigate("/desk");
                return;
              }
              useDeskMarginStore.getState().openMargin(null);
            }}
          >
            <DeskIcon name="plus" size={14} />
          </RailIconButton>
        </div>

        {view === "triage" ? (
          <div className={cn(RAIL_LIST_CLASS, RAIL_TRANSITION_CLASS.triage)}>
            <HoverGlideGroup className="bg-transparent" glideClassName="rounded-[8px] bg-dsk-hover">
              <div className="flex flex-col gap-1">
                {/* Back is where the person came from, the page itself and not only
                    the view, so Triage opened from a case leads back to that case. */}
                <RailPlace
                  to={triageBack.to}
                  current={false}
                  icon="chevL"
                  label={triageBack.view === "cases" ? t("Back to cases") : t("Back to chats")}
                />
                {canWatch && triageBack.view === "chats" && (
                  <RailPlace
                    to="/desk/needs"
                    current={false}
                    icon="stack"
                    label={t("Needs you")}
                    count={lensCounts.everything}
                  />
                )}
              </div>
            </HoverGlideGroup>
            {/* A room's places come through a portal, whose pointer events never
                reach this rail's glide, so the room brings a glide of its own. */}
            <div ref={setSlot} className="flex flex-col" />
          </div>
        ) : (
          <RailList
            className={RAIL_TRANSITION_CLASS[view]}
            rows={rows}
            renderRow={renderRow}
            onLoadMore={onLoadMore}
            onThreadsInView={onThreadsInView}
            nav={
              <>
                {view === "cases" ? (
                  <>
                    <RailPlace to="/desk" current={false} icon="chevL" label={t("Back to chats")} />
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
                    {canDecide && (
                      <RailPlace
                        to="/desk/needs?lens=decisions"
                        current={lens === "decisions" || place === "decisions"}
                        label={t("Decisions")}
                        count={lensCounts.decisions}
                      />
                    )}
                    {canTriage && (
                      <RailPlace
                        to="/desk/triage"
                        current={false}
                        icon="mail"
                        label={t("Triage")}
                        count={triageWaiting}
                      />
                    )}
                  </>
                ) : (
                  <>
                    <RailPlace
                      to="/desk"
                      current={place === "today"}
                      icon="home"
                      label={t("Today")}
                    />
                    {canWatch && (
                      <RailPlace
                        to="/desk/needs"
                        current={false}
                        icon="stack"
                        label={t("Cases")}
                        count={lensCounts.everything}
                      />
                    )}
                    {canTriage && (
                      <RailPlace
                        to="/desk/triage"
                        current={false}
                        icon="mail"
                        label={t("Triage")}
                        count={triageWaiting}
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
                  </>
                )}
              </>
            }
            empty={
              !noConversations || rows.length > 0 ? null : view === "cases" ? (
                <p className="m-0 px-2.5 text-sm/normal text-dsk-subtle">
                  {t(
                    "No cases yet. A record you ask an agent about, or one that needs you, shows here.",
                  )}
                </p>
              ) : (
                <p className="m-0 px-2.5 text-sm/normal text-dsk-subtle">
                  {rt("Your chats will show up here. Press <kbd/> to start one.", {
                    kbd: () => <span className="dk-kbd">{formatModShiftShortcut("O")}</span>,
                  })}
                </p>
              )
            }
          />
        )}

        <div className="flex h-13 flex-none items-center gap-2 border-t border-dsk-b-sub pr-2 pl-3">
          <span className="grid size-6.5 flex-none place-items-center rounded-full border border-dsk-b-sub bg-dsk-sunken text-2xs font-medium text-dsk-muted">
            {getNameInitials(name, "")}
          </span>
          <b className="min-w-0 flex-1 truncate text-dsk-base font-medium" title={name}>
            {name}
          </b>
          <RailIconButton
            label={t("Keyboard shortcuts")}
            hint={t("Keyboard shortcuts (?)")}
            onClick={() => {
              // Triage has keys of its own, and its sheet lists them with the Desk's.
              if (view === "triage") {
                useTriageStore.getState().openOverlay({ kind: "shortcuts" });
                return;
              }
              setDialogOpen("shortcuts", true);
            }}
          >
            <DeskIcon name="keyboard" size={14} />
          </RailIconButton>
          <RailIconButton label={t("Desk settings")} hint={t("Settings")} onClick={onSettings}>
            <DeskIcon name="gear" size={14} />
          </RailIconButton>
        </div>
      </div>
      <RenameConversationDialog
        thread={renaming}
        onRename={onRename}
        onClose={() => setRenaming(null)}
      />
      <DeleteConversationCard
        thread={deletion.asking}
        onConfirm={deletion.confirm}
        onClose={deletion.cancel}
      />
    </aside>
  );
}

/** The kind of record a case's path opens, read back from `/desk/r/<kind>/<id>`. */
function caseKindOf(path: string): string | null {
  const [, desk, r, kind] = path.split("/");

  return desk === "desk" && r === "r" && kind ? decodeURIComponent(kind) : null;
}

/** Where a case conversation's row opens, or null for a conversation about no record. */
function caseRowPath(thread: AssistantThread): string | null {
  const record = caseOfThread(thread);

  return record ? casePath(record.kind, record.id) : null;
}
