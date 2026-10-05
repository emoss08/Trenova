import { AGENT_ACCENTS, resolveAgentIdentity } from "@/components/agent-identity/agent-identity";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { RailDot, RailKnobCard, useRailKnob } from "@/components/desk-chat/rail/rail-parts";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { AssistantMark } from "./assistant-mark";
import { assistantShelfHeading, threadAge, type AssistantThreadList } from "./assistant-history";
import { assistantShelves } from "./thread-grouping";
import { useLiveThreadIds } from "./use-active-turns";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * The full-screen layout's list of conversations, drawn with the Desk rail's
 * pieces: its rows, its dot and the raised card that slides to the open
 * conversation. The search row turns into a field on a click or ⌘K, and Esc
 * clears and closes it. The ones a change is waiting on are listed first with
 * a breathing warm dot; a row's time shows on hover and on the open one.
 */
export function AssistantSidebar({
  threads,
  agentsById,
  activeThreadId,
  searchSignal,
  onOpen,
  onNew,
}: AssistantThreadList & {
  /** Opens the search each time it changes (⌘K). */
  searchSignal: number;
  onOpen: (thread: AssistantThread) => void;
  onNew: () => void;
}) {
  const t = useT();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const liveThreadIds = useLiveThreadIds();
  const [now] = useState(nowInSeconds);
  const [query, setQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);

  const [seenSignal, setSeenSignal] = useState(searchSignal);
  if (seenSignal !== searchSignal) {
    setSeenSignal(searchSignal);
    setSearching(true);
  }

  const shelves = useMemo(
    () =>
      assistantShelves(threads, {
        now,
        timezone,
        query,
        agentName: (thread) => agentsById.get(thread.agentDefinitionId)?.name ?? "",
      }),
    [agentsById, now, query, threads, timezone],
  );
  const activeKey = activeThreadId ? `c:${activeThreadId}` : null;
  const knob = useRailKnob(listRef, activeKey, shelves);

  const closeSearch = () => {
    setQuery("");
    setSearching(false);
  };
  const inputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (searching) {
      inputRef.current?.focus();
    }
  }, [searching, searchSignal]);

  return (
    <div className="as-sbw">
      <aside className="as-sb" aria-label={t("Conversations")}>
        <div className="as-sb-top">
          <span className="as-sb-mk">
            <AssistantMark className="size-3.5" />
          </span>
          <b>{t("Assistant")}</b>
          <button
            type="button"
            className="dk-ib as-sb-new"
            title={t("New conversation")}
            aria-label={t("New conversation")}
            onClick={onNew}
          >
            <DeskIcon name="plus" size={15} />
          </button>
        </div>
        <div className="as-sb-s">
          {searching ? (
            <label className="as-sb-in">
              <DeskIcon name="search" size={13} />
              <input
                ref={inputRef}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    event.stopPropagation();
                    closeSearch();
                  }
                }}
                onBlur={() => query === "" && setSearching(false)}
                placeholder={t("Search conversations")}
                aria-label={t("Search conversations")}
              />
              {query !== "" && (
                <button
                  type="button"
                  aria-label={t("Clear the search")}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={closeSearch}
                >
                  <DeskIcon name="x" size={12} />
                </button>
              )}
            </label>
          ) : (
            <button type="button" className="as-sb-sb" onClick={() => setSearching(true)}>
              <DeskIcon name="search" size={13} />
              <span>{t("Search")}</span>
              <span className="dk-kbd">⌘K</span>
            </button>
          )}
        </div>
        <div className="dk-sb-list" ref={listRef}>
          <RailKnobCard knob={knob} />
          {shelves.map((shelf) => (
            <Fragment key={shelf.key}>
              <div className={cn("dk-sb-gh", shelf.key === "waiting" && "as-w")}>
                {assistantShelfHeading(t, shelf.key)}
              </div>
              {shelf.threads.map((thread) => {
                const agent = agentsById.get(thread.agentDefinitionId);
                const active = thread.id === activeThreadId;
                const state = deskThreadState(thread, {
                  live: liveThreadIds.has(thread.id),
                  active,
                });
                const title = thread.title || t("Untitled conversation");
                return (
                  <button
                    key={thread.id}
                    type="button"
                    data-k={`c:${thread.id}`}
                    className={cn("dk-sb-i dk-sb-c", active && "dk-on")}
                    aria-current={active || undefined}
                    title={[title, agent?.name ?? t("Agent unavailable")].join(" · ")}
                    onClick={() => onOpen(thread)}
                  >
                    <RailDot
                      state={state === "wait" ? null : state}
                      accent={AGENT_ACCENTS[resolveAgentIdentity(agent ?? {}).accent]}
                    />
                    <span className="dk-sb-t">{title}</span>
                    {shelf.key === "waiting" ? (
                      <i className="as-wd" aria-label={t("Waiting on your approval")} />
                    ) : (
                      <em>{threadAge(thread, now)}</em>
                    )}
                  </button>
                );
              })}
            </Fragment>
          ))}
          {shelves.length === 0 && (
            <div className="as-empty">
              {query.trim() === ""
                ? t("Your conversations will show up here.")
                : t("Nothing matches “{0}”", query.trim())}
            </div>
          )}
        </div>
        <div className="as-sb-f">
          <DeskIcon name="lock" size={11} />
          {t("Read-only until you approve")}
        </div>
      </aside>
    </div>
  );
}
