import type { DeskChapterNav } from "@/components/desk-chat/use-chapter-nav";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";
import type { ChapterEntry } from "./chapter-work";
import "./desk-chapters.css";

/** The most ticks the folded rail draws; a longer conversation shows those around the one in view. */
export const RAIL_TICKS = 12;

/** How long the list stays open after the pointer leaves it, so a pass over the gap does not close it. */
const CLOSE_DELAY_MS = 160;

/** The chapters the folded rail draws a tick for: all of them, or a window around the one in view. */
export function tickWindow(
  entries: readonly ChapterEntry[],
  currentIndex: number,
  size = RAIL_TICKS,
): ChapterEntry[] {
  if (entries.length <= size) {
    return [...entries];
  }
  const half = Math.floor(size / 2);
  const start = Math.min(Math.max(0, currentIndex - half), entries.length - size);
  return entries.slice(start, start + size);
}

/**
 * The conversation's chapters in its top-left corner. Folded, it is a tick
 * per chapter, the one in view drawn longer and a chapter with a decision
 * waiting on the person marked; pointing at it or moving focus to it opens
 * the list, where each chapter says what it made and what waits, and picking
 * one scrolls to it. Escape folds it again.
 */
export function DeskChapterRail({
  entries,
  nav,
}: {
  entries: readonly ChapterEntry[];
  nav: DeskChapterNav;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const closing = useRef<number | null>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const listId = useId();

  useEffect(
    () => () => {
      if (closing.current !== null) {
        window.clearTimeout(closing.current);
      }
    },
    [],
  );

  if (entries.length < 2) {
    return null;
  }

  const currentIndex = Math.max(
    0,
    entries.findIndex((entry) => entry.chapter.turnId === nav.current),
  );
  const ticks = tickWindow(entries, currentIndex);

  const keepOpen = () => {
    if (closing.current !== null) {
      window.clearTimeout(closing.current);
      closing.current = null;
    }
    setOpen(true);
  };
  const closeSoon = () => {
    closing.current = window.setTimeout(() => setOpen(false), CLOSE_DELAY_MS);
  };
  const pick = (entry: ChapterEntry) => {
    nav.reveal(entry.chapter.turnId);
    setOpen(false);
  };
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape" && open) {
      event.stopPropagation();
      setOpen(false);
      toggleRef.current?.focus();
    }
  };

  return (
    <nav
      className={cn("dk-chap-rail", open && "dk-chap-rail-open")}
      aria-label={t("Chapters")}
      onMouseEnter={keepOpen}
      onMouseLeave={closeSoon}
      onKeyDown={onKeyDown}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setOpen(false);
        }
      }}
    >
      <Button
        ref={toggleRef}
        variant="bare"
        size="bare"
        className="dk-chap-ticks"
        aria-expanded={open}
        aria-controls={listId}
        aria-label={t("{0, plural, one {# chapter} other {# chapters}}", entries.length)}
        onClick={() => setOpen((was) => !was)}
        onFocus={keepOpen}
      >
        {ticks.map((entry) => (
          <span
            key={entry.chapter.id}
            className={cn(
              "dk-chap-tick",
              entry.chapter.turnId === nav.current && "dk-chap-tick-on",
              entry.work.decisions > 0 && "dk-chap-tick-waits",
              entry.live && "dk-chap-tick-live",
            )}
          />
        ))}
      </Button>
      <ol id={listId} className="dk-chap-list" hidden={!open}>
        {entries.map((entry) => (
          <li key={entry.chapter.id}>
            <Button
              variant="bare"
              size="bare"
              className={cn(
                "dk-chap-row",
                entry.chapter.turnId === nav.current && "dk-chap-row-on",
              )}
              aria-current={entry.chapter.turnId === nav.current ? "location" : undefined}
              onClick={() => pick(entry)}
            >
              <span className="dk-chap-row-n">{entry.number}</span>
              <span className="dk-chap-row-body">
                <span className="dk-chap-row-t">{entry.chapter.title}</span>
                <ChapterWorkLine entry={entry} />
              </span>
            </Button>
          </li>
        ))}
      </ol>
    </nav>
  );
}

function ChapterWorkLine({ entry }: { entry: ChapterEntry }) {
  const t = useT();
  const { artifacts, decisions, waits } = entry.work;
  if (!entry.live && artifacts === 0 && decisions === 0 && waits === 0) {
    return null;
  }

  return (
    <span className="dk-chap-work">
      {entry.live && <span className="dk-chap-w-live">{t("Writing…")}</span>}
      {decisions > 0 && (
        <span className="dk-chap-w-decide">
          {t("{0, plural, one {# to decide} other {# to decide}}", decisions)}
        </span>
      )}
      {waits > 0 && <span>{t("{0, plural, one {# waiting} other {# waiting}}", waits)}</span>}
      {artifacts > 0 && <span>{t("{0, plural, one {# made} other {# made}}", artifacts)}</span>}
    </span>
  );
}
