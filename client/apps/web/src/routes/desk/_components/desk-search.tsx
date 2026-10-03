import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import type { DeskSearchKind, DeskSearchResult } from "@/types/assistant";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useNavigate } from "react-router";
import { DeskArtKindIcon, deskArtKind, deskArtKindName } from "./artifacts/desk-art-kinds";
import { DeskIcon, type DeskIconName } from "./desk-icons";

const SEARCH_DEBOUNCE_MS = 160;

type ResultKind = DeskSearchResult["kind"];

const GROUP_ICONS: Record<ResultKind, DeskIconName> = {
  chat: "chat",
  msg: "copy",
  art: "table",
  dec: "inbox",
};

function groupLabel(kind: ResultKind, t: TranslateFn): string {
  switch (kind) {
    case "chat":
      return t("Conversations");
    case "msg":
      return t("Messages");
    case "art":
      return t("Artifacts");
    case "dec":
      return t("Decisions");
  }
}

const FILTERS: DeskSearchKind[] = ["all", "chat", "msg", "art", "dec"];

function filterLabel(kind: DeskSearchKind, t: TranslateFn): string {
  return kind === "all" ? t("All") : kind === "chat" ? t("Chats") : groupLabel(kind, t);
}

const KIND_ORDER: ResultKind[] = ["chat", "msg", "art", "dec"];

/** The text with every place it matches the query marked. */
function Highlight({ text, query }: { text: string; query: string }) {
  if (query === "") {
    return text;
  }
  const lower = text.toLowerCase();
  const needle = query.toLowerCase();
  const parts: Array<string | React.JSX.Element> = [];
  let at = 0;
  let index = lower.indexOf(needle, at);
  while (index >= 0 && parts.length < 12) {
    if (index > at) {
      parts.push(text.slice(at, index));
    }
    parts.push(<mark key={index}>{text.slice(index, index + needle.length)}</mark>);
    at = index + needle.length;
    index = lower.indexOf(needle, at);
  }
  parts.push(text.slice(at));

  return parts;
}

function relativeWhen(at: number, now: number, t: ReturnType<typeof useT>): string {
  if (at <= 0) {
    return "";
  }
  const minutes = Math.max(0, Math.round((now - at * 1000) / 60_000));
  if (minutes < 1) {
    return t("just now");
  }
  if (minutes < 60) {
    return t("{0}m ago", minutes);
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return t("{0}h ago", hours);
  }
  const days = Math.round(hours / 24);
  if (days === 1) {
    return t("Yesterday");
  }
  if (days < 7) {
    return t("{0}d ago", days);
  }
  return new Date(at * 1000).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * ⌘K: one box over everything a person has in the Desk — their
 * conversations, what was said in them, the artifacts they produced and the
 * decisions they raised. With nothing typed it offers what was recent and
 * the last few things searched for.
 */
export function DeskSearchPalette({
  agentsById,
  onClose,
}: {
  agentsById: ReadonlyMap<string, AgentChoice>;
  onClose: () => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const recentSearches = useDeskSettingsStore((state) => state.recentSearches);
  const rememberSearch = useDeskSettingsStore((state) => state.rememberSearch);
  const [text, setText] = useState("");
  const [filter, setFilter] = useState<DeskSearchKind>("all");
  const [selected, setSelected] = useState(0);
  const [closing, setClosing] = useState(false);
  const [now] = useState(() => Date.now());
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const typed = text.trim();
  const query = useDebounce(typed, SEARCH_DEBOUNCE_MS);
  const searchQuery = useQuery({
    ...queries.assistant.deskSearch(query, filter),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
  });
  const results = useMemo(() => searchQuery.data ?? [], [searchQuery.data]);
  const settled = query === typed && !searchQuery.isPlaceholderData;

  const groups = useMemo(() => {
    const byKind = new Map<ResultKind, DeskSearchResult[]>();
    for (const result of results) {
      byKind.set(result.kind, [...(byKind.get(result.kind) ?? []), result]);
    }
    const kinds = KIND_ORDER.filter((kind) => byKind.has(kind));
    return kinds.map((kind, at) => ({
      kind,
      items: byKind.get(kind) ?? [],
      start: kinds.slice(0, at).reduce((sum, before) => sum + (byKind.get(before)?.length ?? 0), 0),
    }));
  }, [results]);
  const flat = useMemo(() => groups.flatMap((group) => group.items), [groups]);

  // A new query or filter starts the selection over at the top.
  const [seen, setSeen] = useState({ query, filter });
  if (seen.query !== query || seen.filter !== filter) {
    setSeen({ query, filter });
    setSelected(0);
  }

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    const list = listRef.current;
    const row = list?.querySelector<HTMLElement>(`[data-i="${selected}"]`);
    if (!list || !row) {
      return;
    }
    const top = row.offsetTop;
    const bottom = top + row.offsetHeight;
    if (top < list.scrollTop + 32) {
      list.scrollTop = top - 32;
    } else if (bottom > list.scrollTop + list.clientHeight) {
      list.scrollTop = bottom - list.clientHeight + 8;
    }
  }, [selected]);

  const close = useCallback(() => {
    setClosing(true);
    window.setTimeout(onClose, 160);
  }, [onClose]);

  const pick = (result: DeskSearchResult | undefined) => {
    if (!result) {
      return;
    }
    if (query !== "") {
      rememberSearch(query);
    }
    if (result.kind === "dec") {
      void navigate("/desk/decisions");
    } else if (result.kind === "art") {
      void navigate(`${conversationPath(result.threadId)}?a=${result.id}`);
    } else {
      void navigate(conversationPath(result.threadId));
    }
    close();
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    event.stopPropagation();
    if (event.key === "Escape") {
      event.preventDefault();
      close();
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      setSelected((current) => Math.min(flat.length - 1, current + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setSelected((current) => Math.max(0, current - 1));
    } else if (event.key === "Enter") {
      event.preventDefault();
      pick(flat[selected]);
    } else if (event.key === "Tab") {
      event.preventDefault();
      const at = FILTERS.indexOf(filter);
      const step = event.shiftKey ? FILTERS.length - 1 : 1;
      setFilter(FILTERS[(at + step) % FILTERS.length]);
    }
  };

  const meta = (result: DeskSearchResult): string => {
    const agent = agentsById.get(result.agentId)?.name;
    const when = relativeWhen(result.at, now, t);
    switch (result.kind) {
      case "chat":
        return [agent, when].filter(Boolean).join(" · ");
      case "msg":
        return [
          result.status === "User" ? t("You") : agent,
          result.threadTitle || t("Untitled conversation"),
          when,
        ]
          .filter(Boolean)
          .join(" · ");
      case "art":
        return [
          deskArtKindName(deskArtKind({ kind: result.artifactKind ?? "document", payload: {} }), t),
          result.threadTitle || t("Untitled conversation"),
          when,
        ].join(" · ");
      case "dec":
        return [decisionStatus(result.status, t), result.threadTitle, when]
          .filter(Boolean)
          .join(" · ");
    }
  };

  return (
    <div
      className={cn("dk-srch-wrap", closing && "dk-out")}
      onMouseDown={(event) => event.target === event.currentTarget && close()}
    >
      <div className="dk-srch" role="dialog" aria-modal aria-label={t("Search")} onKeyDown={onKeyDown}>
        <div className="dk-srch-in">
          <DeskIcon name="search" size={16} />
          <input
            ref={inputRef}
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder={t("Search chats, messages, artifacts…")}
            aria-label={t("Search the Desk")}
            role="combobox"
            aria-expanded
            aria-controls="dk-srch-list"
            aria-activedescendant={flat[selected] ? `dk-srch-${selected}` : undefined}
          />
          {text ? (
            <button
              type="button"
              className="dk-srch-clr"
              onClick={() => {
                setText("");
                inputRef.current?.focus();
              }}
            >
              {t("Clear")}
            </button>
          ) : (
            <span className="dk-kbd">Esc</span>
          )}
        </div>
        <div className="dk-srch-f" role="tablist" aria-label={t("Search in")}>
          {FILTERS.map((kind) => (
            <button
              key={kind}
              type="button"
              role="tab"
              aria-selected={filter === kind}
              className={filter === kind ? "dk-on" : undefined}
              onClick={() => {
                setFilter(kind);
                inputRef.current?.focus();
              }}
            >
              {filterLabel(kind, t)}
            </button>
          ))}
        </div>
        <div className="dk-srch-l" ref={listRef} id="dk-srch-list" role="listbox">
          {typed === "" && filter === "all" && recentSearches.length > 0 && (
            <div className="dk-srch-rec">
              <div className="dk-srch-gh">{t("Recent searches")}</div>
              <div className="dk-srch-chips">
                {recentSearches.map((recent) => (
                  <button key={recent} type="button" onClick={() => setText(recent)}>
                    <DeskIcon name="replay" size={11} />
                    {recent}
                  </button>
                ))}
              </div>
            </div>
          )}
          {groups.map((group) => (
            <div key={group.kind} role="group" aria-label={groupLabel(group.kind, t)}>
              <div className="dk-srch-gh">
                {query !== "" || filter !== "all"
                  ? groupLabel(group.kind, t)
                  : group.kind === "chat"
                    ? t("Recent conversations")
                    : t("Recent artifacts")}
                <i>{group.items.length}</i>
              </div>
              {group.items.map((result, offset) => {
                const at = group.start + offset;
                return (
                  <button
                    key={`${result.kind}:${result.id}`}
                    id={`dk-srch-${at}`}
                    type="button"
                    role="option"
                    aria-selected={selected === at}
                    data-i={at}
                    className={cn("dk-srch-r", `dk-k-${result.kind}`, selected === at && "dk-on")}
                    onMouseMove={() => selected !== at && setSelected(at)}
                    onClick={() => pick(result)}
                  >
                    <span className="dk-srch-ic">
                      {result.kind === "chat" ? (
                        <span className="dk-sb-dot">
                          <i />
                        </span>
                      ) : result.kind === "art" ? (
                        <DeskArtKindIcon kind={deskArtKind({ kind: result.artifactKind ?? "document", payload: {} })} size={13} />
                      ) : (
                        <DeskIcon name={GROUP_ICONS[result.kind]} size={13} />
                      )}
                    </span>
                    <span className="dk-srch-tx">
                      <span className="dk-srch-t">
                        <Highlight
                          text={result.title || t("Untitled conversation")}
                          query={query}
                        />
                      </span>
                      <span className="dk-srch-m">{meta(result)}</span>
                    </span>
                    <span className="dk-srch-go">
                      <DeskIcon name="enter" size={12} />
                    </span>
                  </button>
                );
              })}
            </div>
          ))}
          {settled && results.length === 0 && (
            <div className="dk-srch-empty">
              {query !== "" ? (
                <>
                  <b>{t("Nothing matches “{0}”", query)}</b>
                  <span>{t("Try a load number, customer or invoice ID.")}</span>
                </>
              ) : (
                <>
                  <b>{t("Nothing here yet")}</b>
                  <span>{t("Conversations and what they produce show up here.")}</span>
                </>
              )}
            </div>
          )}
        </div>
        <div className="dk-srch-ft">
          <span>
            <span className="dk-kbd">↑</span>
            <span className="dk-kbd">↓</span>
            {t("to move")}
          </span>
          <span>
            <span className="dk-kbd">↵</span>
            {t("to open")}
          </span>
          <span>
            <span className="dk-kbd">Tab</span>
            {t("to filter")}
          </span>
          <span className="flex-1" />
          <span aria-live="polite">
            {query !== "" && settled
              ? results.length === 1
                ? t("1 result")
                : t("{0} results", results.length)
              : ""}
          </span>
        </div>
      </div>
    </div>
  );
}

function decisionStatus(status: string, t: TranslateFn): string {
  switch (status) {
    case "Pending":
    case "Ready":
      return t("Waiting on you");
    case "Accepted":
    case "Modified":
    case "Executed":
      return t("Approved");
    case "Rejected":
      return t("Declined");
    case "Expired":
      return t("Expired");
    case "ExecutionFailed":
      return t("Approved, didn't go through");
    default:
      return t("Decision");
  }
}
