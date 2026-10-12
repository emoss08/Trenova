import { useUserTimezone } from "@/hooks/use-user-timezone";
import { AGENT_ACCENTS, resolveAgentIdentity } from "@/components/agent-identity/agent-identity";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { RailDot, RailKnobCard, useRailKnob } from "@/components/desk-chat/rail/rail-parts";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { AssistantIconButton } from "./assistant-icon-button";
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
  const timezone = useUserTimezone();
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
          <AssistantIconButton
            className="as-sb-new bg-dsk-card text-dsk-fg"
            title={t("New conversation")}
            aria-label={t("New conversation")}
            onClick={onNew}
          >
            <DeskIcon name="plus" size={15} />
          </AssistantIconButton>
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
                <Button
                  variant="bare"
                  size="bare"
                  className="size-5 justify-center rounded-md text-dsk-subtle hover:bg-dsk-hover hover:text-dsk-fg"
                  aria-label={t("Clear the search")}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={closeSearch}
                >
                  <DeskIcon name="x" size={12} />
                </Button>
              )}
            </label>
          ) : (
            <Button
              variant="bare"
              size="bare"
              className="flex h-7.5 w-full gap-2.5 rounded-lg pr-1.5 pl-2.5 text-sm text-dsk-subtle transition-colors duration-150 hover:bg-dsk-fg/4 hover:text-dsk-fg"
              onClick={() => setSearching(true)}
            >
              <DeskIcon name="search" size={13} />
              <span className="flex-1 text-left">{t("Search")}</span>
              <span className="dk-kbd">⌘K</span>
            </Button>
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
                  <Button
                    key={thread.id}
                    variant="bare"
                    size="bare"
                    data-k={`c:${thread.id}`}
                    className={cn(
                      "dk-sb-c group/row relative z-1 flex h-7.5 w-full gap-2.5 rounded-lg pr-2 pl-2.5 text-left text-sm text-dsk-muted transition-colors duration-150",
                      active ? "dk-on text-dsk-fg" : "hover:bg-dsk-fg/4 hover:text-dsk-fg",
                    )}
                    aria-current={active || undefined}
                    title={[title, agent?.name ?? t("Agent unavailable")].join(" · ")}
                    onClick={() => onOpen(thread)}
                  >
                    <RailDot
                      state={state === "wait" ? null : state}
                      accent={AGENT_ACCENTS[resolveAgentIdentity(agent ?? {}).accent]}
                    />
                    <span className="dk-sb-t min-w-0 flex-1 truncate">{title}</span>
                    {shelf.key === "waiting" ? (
                      <i
                        className="as-wd mr-0.75 ml-auto"
                        aria-label={t("Waiting on your approval")}
                      />
                    ) : (
                      <em
                        className={cn(
                          "ml-auto font-mono text-xs text-dsk-faint not-italic opacity-0 transition-opacity duration-150 group-hover/row:opacity-100",
                          active && "opacity-100",
                        )}
                      >
                        {threadAge(thread, now)}
                      </em>
                    )}
                  </Button>
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
