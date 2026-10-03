import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  ArtIcon,
  DESK_ART_KINDS,
  DeskArtKindIcon,
  deskArtKind,
  deskArtKindLabel,
  deskArtKindName,
  type DeskArtKind,
} from "./desk-art-kinds";
import { artifactPreview } from "./artifact-preview";
import type { ArtifactLineage } from "./desk-lineage";

/** How many rows the browser draws before it reaches for more. */
const PAGE = 60;

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
 * Every artifact the conversation made, to search: by title, by the tool that
 * made it or by kind, pinned ones on their own, grouped by day and by the turn
 * that made them. Enter opens the first match; Esc clears the search, then
 * goes back.
 */
export function DeskArtifactBrowser({
  lineages,
  activeId,
  onPick,
  onBack,
  onClose,
}: {
  lineages: ArtifactLineage[];
  activeId: string;
  onPick: (id: string) => void;
  onBack: () => void;
  onClose: () => void;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [kind, setKind] = useState<DeskArtKind | "all">("all");
  const [onlyPinned, setOnlyPinned] = useState(false);
  const [limit, setLimit] = useState(PAGE);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => inputRef.current?.focus(), 60);
    return () => window.clearTimeout(timer);
  }, []);

  const needle = query.trim().toLowerCase();
  const base = useMemo(
    () =>
      lineages.filter((lineage) => {
        if (onlyPinned && !lineage.versions.some((version) => version.pinned)) {
          return false;
        }
        if (needle === "") {
          return true;
        }
        const label = deskArtKindLabel(deskArtKind(lineage.latest));
        return `${lineage.latest.title} ${toolOf(lineage)} ${label}`.toLowerCase().includes(needle);
      }),
    [lineages, needle, onlyPinned],
  );
  const counts = useMemo(() => {
    const tally = new Map<DeskArtKind, number>();
    for (const lineage of base) {
      const key = deskArtKind(lineage.latest);
      tally.set(key, (tally.get(key) ?? 0) + 1);
    }
    return tally;
  }, [base]);
  const shown =
    kind === "all" ? base : base.filter((lineage) => deskArtKind(lineage.latest) === kind);
  const page = shown.slice(0, limit);
  const pinnedCount = lineages.filter((lineage) =>
    lineage.versions.some((version) => version.pinned),
  ).length;

  const resetPaging = () => {
    setLimit(PAGE);
    listRef.current?.scrollTo({ top: 0 });
  };

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          setLimit((current) => current + PAGE);
        }
      },
      { root: listRef.current, rootMargin: "200px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [page.length, shown.length]);

  const now = new Date();
  const today = now.toDateString();
  const yesterday = new Date(now.getTime() - 86_400_000).toDateString();
  const groups: Array<{
    day: string;
    turns: Array<{ key: string; at: number; items: ArtifactLineage[] }>;
  }> = [];
  for (const lineage of page) {
    const day = dayLabel(lineage.latest.createdAt, today, yesterday);
    let group = groups.at(-1);
    if (!group || group.day !== day) {
      group = { day, turns: [] };
      groups.push(group);
    }
    const turnKey = lineage.latest.messageId || shortTime(lineage.latest.createdAt);
    let turn = group.turns.at(-1);
    if (!turn || turn.key !== turnKey) {
      turn = { key: turnKey, at: lineage.latest.createdAt, items: [] };
      group.turns.push(turn);
    }
    turn.items.push(lineage);
  }

  return (
    <div className="dk-axb">
      <div className="dk-axb-top">
        <button
          type="button"
          className="dk-ax-ib"
          onClick={onBack}
          title={t("Back")}
          aria-label={t("Back")}
        >
          <ArtIcon name="up" size={13} stroke={2.2} />
        </button>
        <b>{t("All artifacts")}</b>
        <span className="dk-axb-n">{lineages.length}</span>
        <span className="flex-1" />
        <button
          type="button"
          className="dk-ax-ib"
          onClick={onClose}
          title={t("Close")}
          aria-label={t("Hide artifacts")}
        >
          <ArtIcon name="x" size={13} stroke={2.2} />
        </button>
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
        <span className="dk-kbd">⌘J</span>
      </div>
      <div className="dk-axb-f">
        <button
          type="button"
          className={kind === "all" && !onlyPinned ? "dk-on" : undefined}
          onClick={() => {
            setKind("all");
            setOnlyPinned(false);
            resetPaging();
          }}
        >
          {t("All")} <i>{base.length}</i>
        </button>
        <button
          type="button"
          className={onlyPinned ? "dk-on" : undefined}
          onClick={() => {
            setOnlyPinned((value) => !value);
            resetPaging();
          }}
        >
          <ArtIcon name="pin" size={11} />
          {t("Pinned")} <i>{pinnedCount}</i>
        </button>
        {DESK_ART_KINDS.filter((candidate) => counts.has(candidate)).map((candidate) => (
          <button
            key={candidate}
            type="button"
            className={kind === candidate ? "dk-on" : undefined}
            onClick={() => {
              setKind((current) => (current === candidate ? "all" : candidate));
              resetPaging();
            }}
          >
            <span className={cn("dk-ax-ki", `dk-k-${candidate}`)}>
              <DeskArtKindIcon kind={candidate} size={11} />
            </span>
            {deskArtKindName(candidate, t)} <i>{counts.get(candidate)}</i>
          </button>
        ))}
      </div>
      <div className="dk-axb-l" ref={listRef}>
        {groups.map((group) => (
          <section key={group.day}>
            <div className="dk-axb-day">{t(group.day)}</div>
            {group.turns.map((turn) => (
              <div key={turn.key} className="dk-axb-turn">
                <div className="dk-axb-th">
                  <span>{shortTime(turn.at)}</span>
                </div>
                {turn.items.map((lineage) => {
                  const itemKind = deskArtKind(lineage.latest);
                  const pinned = lineage.versions.some((version) => version.pinned);
                  const preview = artifactPreview(lineage.latest, t);
                  return (
                    <Fragment key={lineage.id}>
                      <button
                        type="button"
                        className={cn("dk-axb-r", lineage.id === activeId && "dk-on")}
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
                        <span className="dk-axb-go">
                          <ArtIcon name="ext" size={12} />
                        </span>
                      </button>
                    </Fragment>
                  );
                })}
              </div>
            ))}
          </section>
        ))}
        {page.length < shown.length && (
          <div ref={sentinelRef} className="dk-axb-more">
            <span />
            {t("Loading {0} more…", Math.min(PAGE, shown.length - page.length))}
          </div>
        )}
        {shown.length === 0 && (
          <div className="dk-axb-empty">
            <b>{needle ? t("No artifacts match “{0}”", query) : t("Nothing pinned yet")}</b>
            <span>
              {needle
                ? t("Try a tool name like list_workers or a record number.")
                : t("Pin an artifact to keep it at the top of this conversation.")}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
