"use no memo";
import { deskRailRowKey, type DeskRailRow } from "@/components/desk-chat/rail/desk-rail-rows";
import { RailKnobCard, useRailKnob, type RailKnob } from "@/components/desk-chat/rail/rail-parts";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";

/** Rows from the place the next page arrives at which it is asked for. */
const PREFETCH_ROWS = 15;
const OVERSCAN = 12;
/** A first window tall enough for a rail before the list is measured. */
const INITIAL_HEIGHT = 900;

const ROW_ESTIMATE: Record<DeskRailRow["kind"], number> = {
  shelf: 37,
  thread: 30,
  more: 30,
};

/** Where the rail's conversations come from, a page at a time. */
export type DeskRailPaging = {
  hasMore: boolean;
  loadingMore: boolean;
  /** The last page asked for could not be read; it is asked for again only on request. */
  failed: boolean;
  loadMore: () => void;
};

type DeskRailListProps = {
  /** The `data-k` of the current place or conversation, which the raised card sits behind. */
  activeKey: string;
  /** The places above the conversations: always drawn, never virtualized. */
  nav: ReactNode;
  rows: readonly DeskRailRow[];
  renderRow: (row: DeskRailRow) => ReactNode;
  paging: DeskRailPaging;
  /** Shown below the places when there are no conversations at all. */
  empty: ReactNode;
  /** Told which conversations are drawn, overscan included, and told none when the list goes. */
  onThreadsInView?: (threadIds: readonly string[]) => void;
};

/**
 * The rail's scrolling body: the places, then the conversations, of which
 * only the rows near the viewport are mounted however many have been read.
 * The next page is asked for before the reader reaches the place it arrives.
 *
 * The raised card behind the current row is placed from the virtualizer's
 * measurements rather than from the row's element, so it stays put while
 * that row is scrolled out and unmounted, and slides when rows above it grow
 * or leave.
 */
export function DeskRailList({
  activeKey,
  nav,
  rows,
  renderRow,
  paging,
  empty,
  onThreadsInView,
}: DeskRailListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const navRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const [scrollMargin, setScrollMargin] = useState(0);

  useLayoutEffect(() => {
    const navEl = navRef.current;
    const body = bodyRef.current;
    if (!navEl || !body) {
      return;
    }
    const measure = () => setScrollMargin(body.offsetTop);
    measure();
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(measure);
    observer.observe(navEl);
    return () => observer.disconnect();
  }, []);

  let moreIndex = -1;
  let activeIndex = -1;
  for (let index = 0; index < rows.length; index += 1) {
    const key = deskRailRowKey(rows[index]);
    if (key === activeKey) activeIndex = index;
    if (rows[index].kind === "more") moreIndex = index;
  }

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => ROW_ESTIMATE[rows[index].kind],
    getItemKey: (index) => deskRailRowKey(rows[index]),
    overscan: OVERSCAN,
    scrollMargin,
    initialRect: { width: 0, height: INITIAL_HEIGHT },
  });
  const items = virtualizer.getVirtualItems();
  const totalSize = virtualizer.getTotalSize();
  const lastInView = items.at(-1)?.index ?? -1;

  // Checked again whenever the rows change: a request made while the whole
  // list is being read again is dropped, and the reader may not scroll again.
  const { hasMore, loadingMore, failed, loadMore } = paging;
  useEffect(() => {
    if (
      moreIndex >= 0 &&
      hasMore &&
      !loadingMore &&
      !failed &&
      lastInView >= moreIndex - PREFETCH_ROWS
    ) {
      loadMore();
    }
  }, [failed, hasMore, lastInView, loadMore, loadingMore, moreIndex, rows]);

  const drawnThreads: string[] = [];
  for (const item of items) {
    const row = rows[item.index];
    if (row?.kind === "thread") {
      drawnThreads.push(row.thread.id);
    }
  }
  // Compared as one string, so the list hears about a change in what is drawn
  // rather than about every scroll frame.
  const drawnKey = drawnThreads.join(",");
  useEffect(() => {
    onThreadsInView?.(drawnKey === "" ? [] : drawnKey.split(","));
  }, [drawnKey, onThreadsInView]);
  useEffect(() => () => onThreadsInView?.([]), [onThreadsInView]);

  const navKnob = useRailKnob(scrollRef, activeIndex < 0 ? activeKey : null, scrollMargin);
  const measured = activeIndex >= 0 ? virtualizer.measurementsCache[activeIndex] : undefined;
  const knob: RailKnob | null =
    navKnob ?? (measured ? { y: measured.start, h: measured.size } : null);

  return (
    <div className="dk-sb-list" ref={scrollRef}>
      <RailKnobCard knob={knob} />
      <div ref={navRef}>{nav}</div>
      <div ref={bodyRef} className="relative" style={{ height: totalSize }}>
        {items.map((item) => (
          <div
            key={item.key}
            data-index={item.index}
            ref={virtualizer.measureElement}
            className="absolute inset-x-0 top-0"
            style={{ transform: `translateY(${item.start - scrollMargin}px)` }}
          >
            {renderRow(rows[item.index])}
          </div>
        ))}
      </div>
      {rows.length === 0 && empty}
    </div>
  );
}
