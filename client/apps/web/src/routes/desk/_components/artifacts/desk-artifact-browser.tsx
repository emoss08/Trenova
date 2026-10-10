import { Button } from "@trenova/shared/components/ui/button";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  ArtIcon,
  DESK_ART_KINDS,
  DeskArtKindIcon,
  deskArtKind,
  deskArtKindName,
  type DeskArtKind,
} from "./desk-art-kinds";
import { artifactPreview } from "./artifact-preview";
import { groupLineages, type ArtifactLineage } from "./desk-lineage";
import { NO_PENDING_LOOKUPS, withoutPendingLookups } from "./pending-lookups";

/** How many lineages each page the browser reads holds. */
const PAGE = 60;

function filterChipClass(on: boolean): string {
  return cn(
    "h-6.5 flex-none gap-1.25 rounded-full px-2.25 text-xs text-dsk-muted ring-1 ring-dsk-b-sub ring-inset transition-all duration-120 hover:text-dsk-fg hover:ring-dsk-b [&_i]:font-plex-mono [&_i]:text-xs [&_i]:text-dsk-faint [&_i]:not-italic",
    on && "bg-dsk-ink text-dsk-ink-fg ring-0 hover:text-dsk-ink-fg [&_i]:text-dsk-ink-fg",
  );
}
/** How long the search waits after the last keystroke before it asks. */
const SEARCH_DELAY_MS = 200;

function dayLabel(at: number, today: string, yesterday: string): string {
  const day = new Date(at * 1000).toDateString();
  if (day === today) return "Today";
  if (day === yesterday) return "Yesterday";
  return new Date(at * 1000).toLocaleDateString([], {
    weekday: "long",
    month: "short",
    day: "numeric",
  });
}

function shortTime(at: number): string {
  return new Date(at * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

function highlight(text: string, needle: string): ReactNode {
  if (needle === "") return text;
  const at = text.toLowerCase().indexOf(needle);
  if (at < 0) return text;
  return (
    <>
      {text.slice(0, at)}
      <mark>{text.slice(at, at + needle.length)}</mark>
      {text.slice(at + needle.length)}
    </>
  );
}

function toolOf(lineage: ArtifactLineage): string {
  const tool = lineage.latest.payload.tool;
  return typeof tool === "string" ? tool : "";
}

/**
 * Every artifact the conversation made, to search: by title or by the tool
 * that made it, by kind, pinned ones on their own, grouped by day and by the
 * turn that made them. The search, the kinds and the counts are the server's,
 * a page at a time, so a long conversation is searched whole. Enter opens the
 * first match; Esc clears the search, then goes back.
 */
export function DeskArtifactBrowser({
  threadId,
  total,
  pendingLookups = NO_PENDING_LOOKUPS,
  activeId,
  onPick,
  onBack,
  onClose,
}: {
  threadId: string;
  /** How many lineages the conversation has, from the pane's own read. */
  total: number;
  /** The running turn's lookups, left out until the reply says which it keeps. */
  pendingLookups?: ReadonlySet<string>;
  activeId: string;
  onPick: (id: string) => void;
  onBack: () => void;
  onClose: () => void;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [searched, setSearched] = useState("");
  const [kind, setKind] = useState<DeskArtKind | "all">("all");
  const [onlyPinned, setOnlyPinned] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => inputRef.current?.focus(), 60);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => setSearched(query.trim()), SEARCH_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [query]);

  const filters = { q: searched, kind: kind === "all" ? "" : kind, pinned: onlyPinned };
  const pages = useInfiniteQuery({
    queryKey: queries.assistant.artifacts(threadId)._ctx.browse(filters).queryKey,
    queryFn: ({ pageParam, signal }) =>
      apiService.assistantService.listArtifacts(threadId, {
        signal,
        limit: PAGE,
        cursor: pageParam,
        ...filters,
      }),
    initialPageParam: "",
    getNextPageParam: (page) => page.nextCursor || undefined,
    placeholderData: (previous) => previous,
  });

  const needle = searched.toLowerCase();
  const first = pages.data?.pages[0];
  const visible = useMemo(
    () =>
      withoutPendingLookups(
        pages.data?.pages.flatMap((page) => page.results) ?? [],
        pages.data?.pages[0]?.counts,
        pendingLookups,
      ),
    [pages.data, pendingLookups],
  );
  const shown = useMemo(() => groupLineages(visible.results), [visible.results]);
  const counts = visible.counts;
  const matching = Math.max(0, (first?.total ?? 0) - visible.hidden);
  const remaining = Math.max(0, matching - shown.length);

  const resetPaging = () => {
    listRef.current?.scrollTo({ top: 0 });
  };

  const { fetchNextPage, hasNextPage, isFetchingNextPage } = pages;
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasNextPage) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { root: listRef.current, rootMargin: "200px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage, shown.length]);

  const now = new Date();
  const today = now.toDateString();
  const yesterday = new Date(now.getTime() - 86_400_000).toDateString();
  const groups: Array<{
    day: string;
    turns: Array<{ key: string; at: number; label: string; items: ArtifactLineage[] }>;
  }> = [];
  for (const lineage of shown) {
    const day = dayLabel(lineage.latest.createdAt, today, yesterday);
    let group = groups.at(-1);
    if (!group || group.day !== day) {
      group = { day, turns: [] };
      groups.push(group);
    }
    const turnKey = lineage.latest.messageId || shortTime(lineage.latest.createdAt);
    let turn = group.turns.at(-1);
    if (!turn || turn.key !== turnKey) {
      turn = { key: turnKey, at: lineage.latest.createdAt, label: lineage.latest.turn, items: [] };
      group.turns.push(turn);
    }
    turn.items.push(lineage);
  }

  return (
    <div className="dk-axb">
      <div className="dk-axb-top">
        <Button
          variant="quiet"
          size="icon-sm"
          className="text-dsk-subtle [&_svg]:-rotate-90"
          onClick={onBack}
          title={t("Back")}
          aria-label={t("Back")}
        >
          <ArtIcon name="up" size={13} stroke={2.2} />
        </Button>
        <b>{t("All artifacts")}</b>
        <span className="dk-axb-n">{counts?.all ?? total}</span>
        <span className="sr-only" aria-live="polite">
          {pages.isSuccess && searched !== ""
            ? t("{0, plural, one {# artifact} other {# artifacts}}", matching)
            : ""}
        </span>
        <span className="flex-1" />
        <Button
          variant="quiet"
          size="icon-sm"
          className="text-dsk-subtle"
          onClick={onClose}
          title={t("Close")}
          aria-label={t("Hide artifacts")}
        >
          <ArtIcon name="x" size={13} stroke={2.2} />
        </Button>
      </div>
      <div className="dk-axb-s">
        <ArtIcon name="search" size={14} />
        <input
          ref={inputRef}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            resetPaging();
          }}
          placeholder={t("Search titles, tools, turns…")}
          aria-label={t("Search artifacts")}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.stopPropagation();
              if (query) {
                setQuery("");
              } else {
                onBack();
              }
            }
            if (event.key === "Enter" && shown[0]) {
              onPick(shown[0].id);
            }
          }}
        />
        <span className="dk-kbd" aria-hidden>
          ⌘J
        </span>
      </div>
      <div className="dk-axb-f" role="group" aria-label={t("Filter artifacts")}>
        <Button
          variant="bare"
          size="bare"
          aria-pressed={kind === "all" && !onlyPinned}
          className={filterChipClass(kind === "all" && !onlyPinned)}
          onClick={() => {
            setKind("all");
            setOnlyPinned(false);
            resetPaging();
          }}
        >
          {t("All")} <i>{counts?.all ?? total}</i>
        </Button>
        <Button
          variant="bare"
          size="bare"
          aria-pressed={onlyPinned}
          className={filterChipClass(onlyPinned)}
          onClick={() => {
            setOnlyPinned((value) => !value);
            resetPaging();
          }}
        >
          <ArtIcon name="pin" size={11} />
          {t("Pinned")} <i>{counts?.pinned ?? 0}</i>
        </Button>
        {DESK_ART_KINDS.filter((candidate) => (counts?.families[candidate] ?? 0) > 0).map(
          (candidate) => (
            <Button
              key={candidate}
              variant="bare"
              size="bare"
              aria-pressed={kind === candidate}
              className={filterChipClass(kind === candidate)}
              onClick={() => {
                setKind((current) => (current === candidate ? "all" : candidate));
                resetPaging();
              }}
            >
              <span className={cn("dk-ax-ki", `dk-k-${candidate}`)}>
                <DeskArtKindIcon kind={candidate} size={11} />
              </span>
              {deskArtKindName(candidate, t)} <i>{counts?.families[candidate]}</i>
            </Button>
          ),
        )}
      </div>
      <div className="dk-axb-l" ref={listRef}>
        {groups.map((group) => (
          <section key={group.day}>
            <div className="dk-axb-day">{t(group.day)}</div>
            {group.turns.map((turn) => (
              <div key={turn.key} className="dk-axb-turn">
                <div className="dk-axb-th">
                  <span>
                    {shortTime(turn.at)}
                    {turn.label !== "" && <> · {turn.label}</>}
                  </span>
                </div>
                {turn.items.map((lineage) => {
                  const itemKind = deskArtKind(lineage.latest);
                  const pinned = lineage.versions.some((version) => version.pinned);
                  const preview = artifactPreview(lineage.latest, t);
                  return (
                    <Fragment key={lineage.id}>
                      <Button
                        variant="bare"
                        size="bare"
                        className={cn(
                          "group flex w-full gap-2.75 rounded-lg px-2 py-1.75 text-left transition-colors duration-100 hover:bg-dsk-hover",
                          lineage.id === activeId && "bg-dsk-hover",
                        )}
                        aria-current={lineage.id === activeId ? "true" : undefined}
                        onClick={() => onPick(lineage.id)}
                      >
                        <span className={cn("dk-ax-ki", `dk-k-${itemKind}`)}>
                          <DeskArtKindIcon kind={itemKind} size={14} />
                        </span>
                        <span className="dk-axb-rt">
                          <b>{highlight(lineage.latest.title, needle)}</b>
                          <span>
                            {deskArtKindName(itemKind, t)}
                            {preview !== "" ? (
                              <> · {highlight(preview, needle)}</>
                            ) : (
                              toolOf(lineage) !== "" && <> · {highlight(toolOf(lineage), needle)}</>
                            )}
                          </span>
                        </span>
                        {lineage.versions.length > 1 && (
                          <span className="dk-axb-v">v{lineage.versions.length}</span>
                        )}
                        {pinned && (
                          <span className="dk-axb-pin">
                            <ArtIcon name="pin" size={11} />
                          </span>
                        )}
                        <span className="text-dsk-subtle opacity-0 transition-opacity duration-120 group-hover:opacity-100">
                          <ArtIcon name="ext" size={12} />
                        </span>
                      </Button>
                    </Fragment>
                  );
                })}
              </div>
            ))}
          </section>
        ))}
        {hasNextPage && (
          <div ref={sentinelRef} className="dk-axb-more">
            <span />
            {t("Loading {0} more…", Math.min(PAGE, remaining || PAGE))}
          </div>
        )}
        {pages.isSuccess && shown.length === 0 && (
          <div className="dk-axb-empty">
            <b>
              {needle || kind !== "all"
                ? t("No artifacts match “{0}”", query)
                : t("Nothing pinned yet")}
            </b>
            <span>
              {needle || kind !== "all"
                ? t("Try a tool name like list_shipments or a load ID.")
                : t("Pin an artifact to keep it at the top of this conversation.")}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
