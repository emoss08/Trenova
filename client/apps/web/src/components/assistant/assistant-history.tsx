import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { shelfHeading } from "@/components/desk-chat/rail/rail-parts";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCompactAge, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Fragment, useMemo, useState } from "react";
import { assistantShelves, type AssistantShelfKey } from "./thread-grouping";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/** A shelf's heading in the assistant's lists. */
export function assistantShelfHeading(t: TranslateFn, key: AssistantShelfKey): string {
  return key === "waiting" ? t("Waiting on you") : shelfHeading(t, key);
}

/** When a conversation was last touched, as "now", "4m", "13h" or "2d". */
export function threadAge(thread: AssistantThread, now: number): string {
  const at = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
  return formatCompactAge(now - at);
}

/** The person's conversations, the agents they are with, and the open one. */
export type AssistantThreadList = {
  threads: readonly AssistantThread[];
  agentsById: ReadonlyMap<string, AgentChoice>;
  activeThreadId: string | null;
};

/**
 * Every conversation, searchable, in the compact and side layouts: the ones a
 * change is waiting on first, then today, yesterday and the days before. Each
 * row names its agent and how long ago it was touched; one waiting on the
 * person carries a warm dot.
 */
export function AssistantHistory({
  threads,
  agentsById,
  activeThreadId,
  onOpen,
  onNew,
}: AssistantThreadList & {
  onOpen: (thread: AssistantThread) => void;
  onNew: () => void;
}) {
  const t = useT();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const [now] = useState(nowInSeconds);
  const [query, setQuery] = useState("");
  const agentName = (thread: AssistantThread) =>
    agentsById.get(thread.agentDefinitionId)?.name ?? "";
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

  return (
    <div className="as-hl">
      <div className="as-hs">
        <label className="as-hsi">
          <DeskIcon name="search" size={13} />
          <input
            // oxlint-disable-next-line jsx-a11y/no-autofocus -- the person opened the list to look for one
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape" && query !== "") {
                event.stopPropagation();
                setQuery("");
              }
            }}
            placeholder={t("Search conversations")}
            aria-label={t("Search conversations")}
          />
        </label>
        <button
          type="button"
          className="dk-ib"
          title={t("New conversation")}
          aria-label={t("New conversation")}
          onClick={onNew}
        >
          <DeskIcon name="plus" size={15} />
        </button>
      </div>
      <div className="as-hlist">
        {shelves.map((shelf) => (
          <Fragment key={shelf.key}>
            <div className={cn("as-hg", shelf.key === "waiting" && "as-w")}>
              {assistantShelfHeading(t, shelf.key)}
            </div>
            {shelf.threads.map((thread) => {
              const agent = agentsById.get(thread.agentDefinitionId) ?? null;
              const age = threadAge(thread, now);
              return (
                <button
                  key={thread.id}
                  type="button"
                  className={cn("as-hli", thread.id === activeThreadId && "as-on")}
                  aria-current={thread.id === activeThreadId || undefined}
                  onClick={() => onOpen(thread)}
                >
                  <DeskAgentTile agent={agent} size="xs" />
                  <span>
                    <b>{thread.title || t("Untitled conversation")}</b>
                    <em>
                      {age === "now"
                        ? t("{0} · just now", agentName(thread) || t("Agent unavailable"))
                        : t("{0} · {1} ago", agentName(thread) || t("Agent unavailable"), age)}
                    </em>
                  </span>
                  {shelf.key === "waiting" && (
                    <i className="as-wd" title={t("Waiting on your approval")} />
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
              : t("No conversations match “{0}”", query.trim())}
          </div>
        )}
      </div>
    </div>
  );
}
