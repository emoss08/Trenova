import logo from "@/assets/logo.webp";
import { useLiveThreadIds } from "@/components/assistant/use-active-turns";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  Fragment,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import { useNavigate } from "react-router";
import { DeskIcon } from "./desk-icons";
import { deskThreadState, type DeskThreadState } from "./desk-thread-state";
import { groupDeskThreadsByRecency, type DeskShelfKey } from "./desk-threads";

/** Which of the Desk's places is in front, so the rail can light it. */
export type DeskPlace = "today" | "watchtower" | "decisions" | "thread";

/** How long a deleted row takes to slide out before it leaves the list. */
const ROW_LEAVE_MS = 260;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

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
}: DeskRailProps) {
  const t = useT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const liveThreadIds = useLiveThreadIds();
  const [now] = useState(nowInSeconds);
  const [confirming, setConfirming] = useState<string | null>(null);
  const [leaving, setLeaving] = useState<string | null>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const [knob, setKnob] = useState<{ y: number; h: number } | null>(null);

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
          : "n:today";

  useLayoutEffect(() => {
    const root = listRef.current;
    if (!root) {
      return;
    }
    const element = root.querySelector<HTMLElement>(`[data-k="${CSS.escape(activeKey)}"]`);
    if (!element) {
      setKnob(null);
      return;
    }
    const rootBox = root.getBoundingClientRect();
    const box = element.getBoundingClientRect();
    setKnob({ y: box.top - rootBox.top + root.scrollTop, h: box.height });
  }, [activeKey, shelves]);

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
        <button
          type="button"
          className="dk-ib"
          title={t("Search  ⌘K")}
          aria-label={t("Search")}
          onClick={onSearch}
        >
          <DeskIcon name="search" size={15} />
        </button>
        <button
          type="button"
          className="dk-ib dk-sb-new"
          title={t("New conversation  ⌘N")}
          aria-label={t("New conversation")}
          onClick={() => void navigate("/desk")}
        >
          <DeskIcon name="plus" size={15} stroke={2} />
        </button>
      </div>

      <div className="dk-sb-list" ref={listRef}>
        {knob && (
          <span
            className="dk-sb-knob"
            style={{ transform: `translateY(${knob.y}px)`, height: knob.h }}
          />
        )}
        <button
          type="button"
          data-k="n:today"
          className={cn("dk-sb-i", activeKey === "n:today" && "dk-on")}
          onClick={() => void navigate("/desk")}
        >
          <DeskIcon name="home" size={14} />
          <span>{t("Today")}</span>
        </button>
        {canWatch && (
          <button
            type="button"
            data-k="n:watch"
            className={cn("dk-sb-i", activeKey === "n:watch" && "dk-on")}
            onClick={() => void navigate("/desk/watchtower")}
          >
            <DeskIcon name="radar" size={14} />
            <span>{t("Watchtower")}</span>
            {watchtowerCount > 0 && <em className="dk-sb-ct">{watchtowerCount}</em>}
          </button>
        )}
        {canDecide && (
          <button
            type="button"
            data-k="n:dec"
            className={cn("dk-sb-i", activeKey === "n:dec" && "dk-on")}
            onClick={() => void navigate("/desk/decisions")}
          >
            <DeskIcon name="inbox" size={14} />
            <span>{t("Decisions")}</span>
            {decisionsCount > 0 && (
              <em className={cn("dk-sb-ct", decisionsWaitHere && "dk-w")}>{decisionsCount}</em>
            )}
          </button>
        )}

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
                  )}
                  title={[agent?.name ?? t("Agent unavailable"), stateLabel(t, state)]
                    .filter(Boolean)
                    .join(" · ")}
                  onClick={() => open(thread)}
                  onKeyDown={(event) => openOnKey(event, thread)}
                  onMouseLeave={() => confirming === thread.id && setConfirming(null)}
                >
                  <span className={cn("dk-sb-dot", state && `dk-s-${state}`)}>
                    {state === "wait" ? (
                      <DeskIcon name="info" size={13} stroke={2} />
                    ) : state === "error" ? (
                      <DeskIcon name="alert" size={13} stroke={2} />
                    ) : (
                      <i />
                    )}
                  </span>
                  <span className="dk-sb-t">{title}</span>
                  <span
                    className="dk-sb-acts"
                    onClick={(event) => event.stopPropagation()}
                    onKeyDown={(event) => event.stopPropagation()}
                  >
                    {confirming === thread.id ? (
                      <button
                        type="button"
                        className="dk-sb-a dk-del dk-cf"
                        onClick={() => remove(thread)}
                      >
                        {t("Delete")}
                      </button>
                    ) : (
                      <>
                        <button
                          type="button"
                          className={cn("dk-sb-a", thread.pinned && "dk-pinned")}
                          data-tip={thread.pinned ? t("Unpin") : t("Pin")}
                          aria-label={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
                          aria-pressed={thread.pinned}
                          onClick={() => onTogglePin(thread)}
                        >
                          <DeskIcon name="pin" size={13} />
                        </button>
                        <button
                          type="button"
                          className="dk-sb-a dk-del"
                          data-tip={t("Delete")}
                          aria-label={t("Delete conversation")}
                          onClick={() => remove(thread)}
                        >
                          <DeskIcon name="trash" size={13} />
                        </button>
                      </>
                    )}
                  </span>
                </div>
              );
            })}
          </Fragment>
        ))}
      </div>

      <div className="dk-sb-me">
        <span className="dk-me">{getNameInitials(name, "")}</span>
        <b>{name}</b>
        <span style={{ flex: 1 }} />
        <button
          type="button"
          className="dk-ib"
          title={t("Settings")}
          aria-label={t("Desk settings")}
          onClick={onSettings}
        >
          <DeskIcon name="gear" size={15} />
        </button>
        <button
          type="button"
          className="dk-ib"
          title={t("Back to Trenova")}
          aria-label={t("Back to Trenova")}
          onClick={() => void navigate("/")}
        >
          <DeskIcon name="chevL" size={14} />
        </button>
      </div>
    </aside>
  );
}

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

function stateLabel(t: TranslateFn, state: DeskThreadState): string {
  switch (state) {
    case "work":
      return t("Working");
    case "wait":
      return t("Needs your approval");
    case "error":
      return t("Last reply failed");
    case "new":
      return t("New reply");
    default:
      return "";
  }
}
