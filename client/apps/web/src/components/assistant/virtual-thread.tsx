import { useT } from "@trenova/shared/i18n/use-t";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { cn } from "@trenova/shared/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDownIcon } from "lucide-react";
import { useReducedMotion } from "motion/react";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

export type VirtualThreadRow = {
  key: string;
  render: () => ReactNode;
};

/** How far from the end still counts as reading the latest message. */
const END_THRESHOLD = 96;
/** How many rows from the top trigger the next page of history. */
const LOAD_AHEAD_ROWS = 4;
/** A guess at a row's height before it is measured; a short reply. */
const ESTIMATED_ROW_HEIGHT = 88;

type VirtualThreadProps = {
  rows: readonly VirtualThreadRow[];
  hasOlder: boolean;
  isLoadingOlder: boolean;
  onLoadOlder: () => void;
  /** Space kept clear at the bottom for whatever floats over the thread. */
  paddingBottom: number;
  /**
   * Where the jump-to-latest control sits, from the bottom: the top of the
   * box that floats over the thread. It sits just above that edge, in the
   * band where the thread already fades, so it never covers a line the
   * reader is reading. Defaults to just above the cleared space.
   */
  jumpOffset?: number;
  className?: string;
  contentClassName?: string;
  /** Applied to every row: the gap between messages lives here. */
  rowClassName?: string;
  children?: ReactNode;
};

/**
 * The thread, windowed.
 *
 * Only the rows near the viewport are in the DOM, so a conversation of a few
 * hundred lookups costs what a short one does. The list is anchored to its
 * end: a page of older history arriving above keeps the reader's place, and a
 * reply growing at the bottom follows only while they were already reading
 * the latest one. Scrolling up never fights the stream; a small control
 * offers the way back down instead.
 */
export function VirtualThread({
  rows,
  hasOlder,
  isLoadingOlder,
  onLoadOlder,
  paddingBottom,
  jumpOffset,
  className,
  contentClassName,
  rowClassName,
  children,
}: VirtualThreadProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [atEnd, setAtEnd] = useState(true);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ESTIMATED_ROW_HEIGHT,
    getItemKey: (index) => rows[index].key,
    overscan: 8,
    anchorTo: "end",
    followOnAppend: reduceMotion ? true : "smooth",
    scrollEndThreshold: END_THRESHOLD,
  });

  const items = virtualizer.getVirtualItems();
  const firstIndex = items[0]?.index;

  // The next page is asked for a few rows before the top is reached, so a
  // steady scroll never lands on a blank.
  useEffect(() => {
    if (hasOlder && !isLoadingOlder && firstIndex !== undefined && firstIndex < LOAD_AHEAD_ROWS) {
      onLoadOlder();
    }
  }, [firstIndex, hasOlder, isLoadingOlder, onLoadOlder]);

  const onScroll = useCallback(() => {
    setAtEnd(virtualizer.isAtEnd(END_THRESHOLD));
  }, [virtualizer]);

  // A new row while the reader is at the end keeps them there by the
  // virtualizer's own following; the button only has to know when they left.
  useEffect(() => {
    setAtEnd(virtualizer.isAtEnd(END_THRESHOLD));
  }, [rows.length, virtualizer]);

  // The virtualizer follows an appended row only when the last key changes,
  // and while a turn is live the last row is always the same one. A proposal
  // card landing inside an entry, a message arriving above the live turn, or
  // the reply itself growing all change the height and not the key. While
  // the reader was at the end, growth keeps them there.
  const atEndRef = useRef(true);
  atEndRef.current = atEnd;
  const totalSize = virtualizer.getTotalSize();
  const lastSizeRef = useRef(totalSize);
  useEffect(() => {
    const grew = totalSize > lastSizeRef.current;
    lastSizeRef.current = totalSize;
    if (grew && atEndRef.current) {
      virtualizer.scrollToEnd({ behavior: "auto" });
    }
  }, [totalSize, rows.length, virtualizer]);

  const jumpToLatest = useCallback(() => {
    virtualizer.scrollToEnd({ behavior: reduceMotion ? "auto" : "smooth" });
  }, [reduceMotion, virtualizer]);

  return (
    <div className={cn("relative min-h-0 flex-1", className)}>
      <div
        ref={scrollRef}
        onScroll={onScroll}
        className="scrollbar-overlay size-full min-h-0 overflow-y-auto overscroll-contain"
      >
        <div className={cn("mx-auto w-full", contentClassName)}>
          {/* A fixed slot rather than a spinner that appears: a header that
              changes height would move the rows below it under the pointer. */}
          {hasOlder && (
            <div
              className="text-muted-foreground flex h-8 items-center justify-center text-xs"
              aria-live="polite"
            >
              {isLoadingOlder && (
                <span className="flex items-center gap-1.5">
                  <Spinner className="size-3" />
                  {t("Loading earlier messages…")}
                </span>
              )}
            </div>
          )}
          {children}
          {/* The room under the last row is a block of its own. Padding on
              the sized box counted inside its height, so the rows overran it
              and the last one sat under the composer's fade. */}
          <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
            {items.map((item) => (
              <div
                key={item.key}
                data-index={item.index}
                ref={virtualizer.measureElement}
                className={cn("absolute inset-x-0 top-0", rowClassName)}
                style={{ transform: `translateY(${item.start}px)` }}
              >
                {rows[item.index].render()}
              </div>
            ))}
          </div>
          <div aria-hidden style={{ height: paddingBottom }} />
        </div>
      </div>

      {/* A small pill that floats over the thread, so it is the one thing
          here allowed a lift. It rises into view when the reader leaves the
          end and sinks away when they return; it does not move otherwise. */}
      <div
        className={cn(
          "pointer-events-none absolute inset-x-0 z-20 flex justify-center transition-[opacity,translate] duration-200 ease-settle",
          atEnd ? "translate-y-1 opacity-0" : "translate-y-0 opacity-100",
        )}
        style={{ bottom: jumpOffset ?? paddingBottom + 12 }}
      >
        <button
          type="button"
          className={cn(
            "ui-lift-whisper ui-focus-ring ui-press bg-raised text-foreground-muted hover:text-foreground flex h-6 items-center gap-1 rounded-full px-2.5 text-xs transition-colors",
            atEnd ? "pointer-events-none" : "pointer-events-auto",
          )}
          tabIndex={atEnd ? -1 : 0}
          aria-hidden={atEnd}
          onClick={jumpToLatest}
        >
          <ArrowDownIcon className="size-3" />
          {t("Latest")}
        </button>
      </div>
    </div>
  );
}
