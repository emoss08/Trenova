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
import { Fragment, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useNavigate } from "react-router";
import { caseRailLabel, caseStateLabel } from "./case/case-labels";
import { nextWake, useSnoozeClock } from "./case/use-snooze-clock";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { groupDeskThreadsByRecency } from "@/components/desk-chat/rail/desk-threads";
import {
  RailDot,
  RailKnobCard,
  railTime,
  shelfHeading,
  stateLabel,
  useRailKnob,
} from "@/components/desk-chat/rail/rail-parts";
import { useRichT } from "@trenova/shared/i18n/rich";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace = "today" | "watchtower" | "decisions" | "memory" | "thread" | "agent";

/** How long a deleted row takes to slide out before it leaves the list. */
const ROW_LEAVE_MS = 260;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

const NAV_ITEM_CLASS =
  "relative z-1 flex h-7.5 w-full gap-2.5 rounded-lg pr-2 pl-2.5 text-left text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-fg/4 hover:text-dsk-fg";
const NAV_ITEM_ON_CLASS = "text-dsk-fg hover:bg-transparent";
const NAV_LABEL_CLASS = "min-w-0 flex-1 truncate";
const ROW_ACTION_CLASS =
  "dk-sb-a relative size-5.5 justify-center rounded-md text-dsk-subtle transition-colors duration-150 hover:bg-dsk-fg/8 hover:text-dsk-fg";

export type DeskRailProps = {
  place: DeskPlace;
  threads: readonly AssistantThread[];
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
  /** Renames a conversation from its row: double-click, type, Enter. */
  onRename: (thread: AssistantThread, title: string) => void;
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
 */
export function DeskRail({
  place,
  threads,
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
  onRename,
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
  const [editing, setEditing] = useState<string | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const shelves = useMemo(
    () => groupDeskThreadsByRecency(threads, now, { pinnedFirst: true, timezone }),
    [now, threads, timezone],
  );

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

  const knob = useRailKnob(listRef, activeKey, shelves);

  const remove = (thread: AssistantThread) => {
    if (confirming !== thread.id) {
      setConfirming(thread.id);
      return;
    }
    setConfirming(null);
    setLeaving(thread.id);
    window.setTimeout(() => {
      setLeaving(null);
      onDelete(thread);
    }, ROW_LEAVE_MS);
  };

  const open = (thread: AssistantThread) => void navigate(conversationPath(thread.id));
  const rename = (thread: AssistantThread, value: string) => {
    setEditing(null);
    const title = value.trim();
    if (title !== "" && title !== thread.title) {
      onRename(thread, title);
    }
  };
  const openOnKey = (event: KeyboardEvent, thread: AssistantThread) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      open(thread);
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

      <div className="dk-sb-list" ref={listRef}>
        <RailKnobCard knob={knob} />
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

        {shelves.map((shelf) => (
          <Fragment key={shelf.key}>
            <div className="dk-sb-gh">{shelfHeading(t, shelf.key)}</div>
            {shelf.threads.map((thread) => {
              const agent = agentsById.get(thread.agentDefinitionId);
              const active = thread.id === activeThreadId && place === "thread";
              const state = deskThreadState(thread, {
                live: liveThreadIds.has(thread.id),
                active,
              });
              const title = thread.title || t("Untitled conversation");
              const caseLabel = thread.case ? caseRailLabel(thread.case, now, timezone, t) : "";
              return (
                <div
                  key={thread.id}
                  role="button"
                  tabIndex={0}
                  data-k={`c:${thread.id}`}
                  className={cn(
                    "dk-sb-i dk-sb-c",
                    active && "dk-on",
                    leaving === thread.id && "dk-out",
                    confirming === thread.id && "dk-cf",
                    editing === thread.id && "dk-ed",
                  )}
                  title={[
                    agent?.name ?? t("Agent unavailable"),
                    stateLabel(t, state),
                    thread.case ? caseStateLabel(thread.case, now, timezone, t) : "",
                    railTime(thread, now, timezone),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                  onClick={() => editing !== thread.id && open(thread)}
                  onDoubleClick={() => {
                    setConfirming(null);
                    setEditing(thread.id);
                  }}
                  onKeyDown={(event) => editing !== thread.id && openOnKey(event, thread)}
                  onMouseLeave={() => confirming === thread.id && setConfirming(null)}
                >
                  <RailDot state={state} />
                  {editing === thread.id ? (
                    <input
                      className="dk-sb-in"
                      // oxlint-disable-next-line jsx-a11y/no-autofocus -- the row turned into this field on purpose
                      autoFocus
                      defaultValue={thread.title}
                      aria-label={t("Conversation name")}
                      onFocus={(event) => event.target.select()}
                      onClick={(event) => event.stopPropagation()}
                      onBlur={(event) => rename(thread, event.target.value)}
                      onKeyDown={(event) => {
                        event.stopPropagation();
                        if (event.key === "Enter") rename(thread, event.currentTarget.value);
                        if (event.key === "Escape") setEditing(null);
                      }}
                    />
                  ) : (
                    <span className="dk-sb-t">{title}</span>
                  )}
                  {caseLabel !== "" && editing !== thread.id && (
                    <span className="dk-sb-cs">{caseLabel}</span>
                  )}
                  <span
                    className="dk-sb-acts"
                    onClick={(event) => event.stopPropagation()}
                    onKeyDown={(event) => event.stopPropagation()}
                  >
                    {confirming === thread.id ? (
                      <Button
                        variant="bare"
                        size="bare"
                        className="h-5.5 animate-[dk-pop_180ms_var(--dk-spring)_both] rounded-md bg-danger px-2 text-xs font-medium text-dsk-on-solid transition-colors duration-150 hover:bg-danger-hover"
                        onClick={() => remove(thread)}
                      >
                        {t("Delete")}
                      </Button>
                    ) : (
                      <>
                        <Button
                          variant="bare"
                          size="bare"
                          className={cn(
                            ROW_ACTION_CLASS,
                            thread.pinned && "text-dsk-fg [&_svg_path]:fill-current",
                          )}
                          data-tip={thread.pinned ? t("Unpin") : t("Pin")}
                          aria-label={
                            thread.pinned ? t("Unpin conversation") : t("Pin conversation")
                          }
                          aria-pressed={thread.pinned}
                          onClick={() => onTogglePin(thread)}
                        >
                          <DeskIcon name="pin" size={13} />
                        </Button>
                        <Button
                          variant="bare"
                          size="bare"
                          className={cn(ROW_ACTION_CLASS, "hover:text-dsk-error")}
                          data-tip={t("Delete")}
                          aria-label={t("Delete conversation")}
                          onClick={() => remove(thread)}
                        >
                          <DeskIcon name="trash" size={13} />
                        </Button>
                      </>
                    )}
                  </span>
                </div>
              );
            })}
          </Fragment>
        ))}
        {shelves.length === 0 && (
          <>
            <div className="dk-sb-gh">{t("Conversations")}</div>
            <div className="dk-sb-empty">
              {rt("Your chats will show up here. Press <kbd/> to start one.", {
                kbd: () => <span className="dk-kbd">⌘N</span>,
              })}
            </div>
          </>
        )}
      </div>

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
