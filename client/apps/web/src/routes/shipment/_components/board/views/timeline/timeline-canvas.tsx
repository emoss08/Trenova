"use no memo";

import { usePointerHover } from "@/components/pointer-hover-card";
import { ScrollArea, ScrollBar } from "@trenova/shared/components/ui/scroll-area";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, formatUnixTime, formatUnixWeekday } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { defaultRangeExtractor, useVirtualizer, type Range } from "@tanstack/react-virtual";
import { useEffect, useRef, useState, type CSSProperties, type ReactNode, type Ref } from "react";
import {
  AXIS_HEIGHT,
  GROUP_HEIGHT,
  LABEL_WIDTH,
  NIGHT_BANDS,
  type TimelineScale,
} from "./timeline-scale";

const HOUR = 3_600;
/** How far left of now the sheet opens, so the hours just gone stay in view. */
const NOW_LEAD_PX = 160;
/** Rows kept mounted past each edge of the view, so a fast scroll never shows a gap. */
const OVERSCAN = 8;

/**
 * One line of the sheet: a group header or a row, at its exact height. The
 * sheet mounts only the lines in view, so `node` is an element that costs
 * nothing until it is drawn.
 */
export type TimelineSheetItem = {
  key: string;
  height: number;
  /** A group header: it stays pinned under the axis while its group scrolls by. */
  header?: boolean;
  node: ReactNode;
};

/**
 * Where the sheet was scrolled across, kept by the timeline so switching
 * between shipments and drivers & carriers stays at the same hours. It holds
 * one range: a new range opens on now again.
 */
export type TimelineScrollMemory = {
  recall: (rangeStart: number) => number | null;
  remember: (rangeStart: number, left: number) => void;
};

export function createTimelineScrollMemory(): TimelineScrollMemory {
  let kept: { rangeStart: number; left: number } | null = null;
  return {
    recall: (rangeStart) => (kept?.rangeStart === rangeStart ? kept.left : null),
    remember: (rangeStart, left) => {
      kept = { rangeStart, left };
    },
  };
}

type TimelineCanvasProps = {
  scale: TimelineScale;
  today: number;
  labelHeading: string;
  items: readonly TimelineSheetItem[];
  /** Drawn under the axis when there are no items. */
  empty?: ReactNode;
  /** The key of a line to bring into view when it changes, such as the selected load's row. */
  revealKey?: string | null;
  scrollMemory?: TimelineScrollMemory;
};

/**
 * The scrolling sheet every row sits on: a sticky time axis, night hours and
 * day boundaries behind the rows, and the now line above everything.
 *
 * It is virtualized by row: of a range of hundreds of loads, only the lines in
 * view (and OVERSCAN past each edge) are mounted, each placed at its exact
 * offset, since every line's height is known. The current group's header stays
 * mounted and pinned under the axis; the next one slides over it as it arrives.
 * Columns are not virtualized: a row holds one bar, a lane a few.
 *
 * `"use no memo"`: TanStack Virtual's virtualizer is a mutable object, and the
 * now line is moved by the scroll handler directly.
 */
export function TimelineCanvas({
  scale,
  today,
  labelHeading,
  items,
  empty,
  revealKey = null,
  scrollMemory,
}: TimelineCanvasProps) {
  // The canvas reads the clock itself: a minute passing moves the now line only.
  const now = useNow("minute");
  const hover = usePointerHover();
  const [viewport, setViewport] = useState<HTMLDivElement | null>(null);
  const nowRef = useRef<HTMLDivElement>(null);
  const pinnedRef = useRef<number | null>(null);
  const nowVisible = now > scale.start && now < scale.end;

  const headers: number[] = [];
  items.forEach((item, index) => {
    if (item.header) headers.push(index);
  });

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => viewport,
    estimateSize: (index) => items[index].height,
    getItemKey: (index) => items[index].key,
    overscan: OVERSCAN,
    // Lines are measured from the axis's bottom edge, not the sheet's top, so the
    // first line in range is the first one below the axis: a header pins exactly
    // as it reaches the axis. Lines are drawn AXIS_HEIGHT lower to match.
    // A revealed line lands below its group's pinned header, and clear of the
    // bottom edge the axis's offset would otherwise hide it behind.
    scrollPaddingStart: GROUP_HEIGHT,
    scrollPaddingEnd: AXIS_HEIGHT,
    rangeExtractor: (range: Range) => {
      // The header of the group the view starts in, kept mounted to be pinned.
      let pinned: number | null = null;
      for (const index of headers) {
        if (index <= range.startIndex) pinned = index;
        else break;
      }
      pinnedRef.current = pinned;
      const indexes = defaultRangeExtractor(range);
      return pinned === null || indexes.includes(pinned) ? indexes : [pinned, ...indexes];
    },
  });

  const clipNow = () => {
    const line = nowRef.current;
    if (!line || !viewport) return;
    line.style.visibility = scale.x(now) - viewport.scrollLeft < 0 ? "hidden" : "visible";
  };

  // Scrolling moves the bars out from under the hover card, so the card closes.
  useEffect(() => {
    if (!viewport) return;
    const onScroll = () => {
      clipNow();
      hover?.dismiss();
      scrollMemory?.remember(scale.start, viewport.scrollLeft);
    };
    viewport.addEventListener("scroll", onScroll, { passive: true });
    return () => viewport.removeEventListener("scroll", onScroll);
    // clipNow reads only these.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewport, hover, now, scale, scrollMemory]);

  // Open where this range was last scrolled to, else on now when it is in view,
  // else at the start of the range.
  useEffect(() => {
    if (!viewport) return;
    viewport.scrollLeft =
      scrollMemory?.recall(scale.start) ??
      (nowVisible ? Math.max(0, scale.x(now) - NOW_LEAD_PX) : 0);
    clipNow();
    // Only a new range moves the view; the minute ticking over does not.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewport, scale.start, scale.end]);

  useEffect(() => {
    if (!revealKey) return;
    const index = items.findIndex((item) => item.key === revealKey);
    if (index !== -1) virtualizer.scrollToIndex(index, { align: "auto" });
    // Only a newly revealed line scrolls; a refetch of the same one does not.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [revealKey]);

  const ticks: number[] = [];
  const bands: { left: number; width: number }[] = [];
  for (let d = 0; d < scale.days.length - 1; d++) {
    const dayStart = scale.days[d];
    const dayEnd = scale.days[d + 1];
    for (let h = scale.tickHours; dayStart + h * HOUR < dayEnd; h += scale.tickHours) {
      ticks.push(dayStart + h * HOUR);
    }
    for (const [from, to] of NIGHT_BANDS) {
      const left = scale.x(dayStart + from * HOUR);
      bands.push({ left, width: scale.x(Math.min(dayStart + to * HOUR, dayEnd)) - left });
    }
  }

  // The range extractor picks the pinned header as the virtual items are read.
  const virtualItems = virtualizer.getVirtualItems();
  const pinned = pinnedRef.current;
  const sheetHeight = AXIS_HEIGHT + virtualizer.getTotalSize();

  return (
    <div className="absolute inset-0" style={{ "--tl-label": `${LABEL_WIDTH}px` } as CSSProperties}>
      <ScrollArea
        className="size-full"
        viewportClassName="@container overscroll-contain"
        viewportRef={setViewport}
        // The fades start under the axis and the group header pinned beneath it
        // (AXIS_HEIGHT + GROUP_HEIGHT = 70px), and right of the label column
        // (LABEL_WIDTH), so none of them fades out.
        maskClassName="before:top-[70px] before:left-55"
        type="hover"
      >
        <div
          className="relative"
          style={{ width: LABEL_WIDTH + scale.width, height: sheetHeight, minHeight: "100%" }}
        >
          <div
            aria-hidden
            className="pointer-events-none absolute inset-y-0"
            style={{ left: LABEL_WIDTH }}
          >
            {bands.map((band) => (
              <span
                key={band.left}
                className="bg-sunken absolute inset-y-0"
                style={{ left: band.left, width: band.width }}
              />
            ))}
            {ticks.map((tick) => (
              <span
                key={tick}
                className="bg-border-subtle absolute inset-y-0 w-px"
                style={{ left: scale.x(tick) }}
              />
            ))}
            {scale.days.slice(1, -1).map((day) => (
              <span
                key={day}
                className="bg-border-strong absolute inset-y-0 w-px"
                style={{ left: scale.x(day) }}
              />
            ))}
          </div>

          <TimelineAxis scale={scale} today={today} labelHeading={labelHeading} ticks={ticks} />

          {items.length === 0 ? empty : null}

          {virtualItems.map((row) => {
            const item = items[row.index];
            if (row.index === pinned) {
              // In flow right under the axis, so it sticks there while its group scrolls.
              return (
                <div
                  key={item.key}
                  className="sticky z-7"
                  style={{ top: AXIS_HEIGHT, height: item.height, marginBottom: -item.height }}
                >
                  {item.node}
                </div>
              );
            }
            return (
              <div
                key={item.key}
                className={cn("absolute inset-x-0", item.header && "z-8")}
                style={{ top: AXIS_HEIGHT + row.start, height: item.height }}
              >
                {item.node}
              </div>
            );
          })}

          {nowVisible ? (
            <div
              ref={nowRef}
              aria-hidden
              className="bg-danger pointer-events-none absolute inset-y-0 z-12 w-0.5"
              style={{ left: LABEL_WIDTH + scale.x(now) - 1 }}
            >
              <span className="bg-danger text-foreground-on-solid rounded-badge sticky top-1 ml-px inline-block -translate-x-1/2 px-1 font-mono text-2xs tabular-nums">
                {formatUnixTime(now)}
              </span>
            </div>
          ) : null}
        </div>
        <ScrollBar orientation="horizontal" className="z-30" />
      </ScrollArea>
    </div>
  );
}

function TimelineAxis({
  scale,
  today,
  labelHeading,
  ticks,
}: {
  scale: TimelineScale;
  today: number;
  labelHeading: string;
  ticks: number[];
}) {
  const t = useT();
  const showHours = scale.range !== "week";

  return (
    <div
      className="bg-card border-subtle sticky top-0 z-10 grid border-b"
      style={{ gridTemplateColumns: `${LABEL_WIDTH}px ${scale.width}px`, height: AXIS_HEIGHT }}
    >
      <div className="bg-card border-subtle text-muted-foreground sticky left-0 z-9 flex items-end border-r px-2.5 pb-1.5 text-xs font-medium">
        {labelHeading}
      </div>
      <div className="relative">
        {scale.days.slice(0, -1).map((day) => {
          const isToday = day === today;
          return (
            <span
              key={day}
              className={cn(
                "absolute top-1 pl-1.5 text-xs font-medium whitespace-nowrap",
                isToday ? "text-brand" : "text-foreground",
              )}
              style={{ left: scale.x(day) }}
            >
              {isToday
                ? t("{0}, {1} · today", formatUnixWeekday(day), formatUnixMonthDay(day))
                : t("{0}, {1}", formatUnixWeekday(day), formatUnixMonthDay(day))}
            </span>
          );
        })}
        {showHours
          ? ticks.map((tick) => (
              <span
                key={tick}
                className="text-muted-foreground absolute bottom-1.5 -translate-x-1/2 font-mono text-2xs whitespace-nowrap tabular-nums"
                style={{ left: scale.x(tick) }}
              >
                {formatUnixTime(tick)}
              </span>
            ))
          : null}
      </div>
    </div>
  );
}

type TimelineRowProps = {
  height: number;
  label: ReactNode;
  lane: ReactNode;
  scale: TimelineScale;
  selected?: boolean;
  highlighted?: boolean;
  /** Set when the label opens the row's load. */
  onLabelClick?: () => void;
  labelTitle?: string;
  laneRef?: Ref<HTMLDivElement>;
  dropState?: "ok" | "refused" | null;
};

/**
 * One row: a sticky label and its lane. The label's fills are all opaque,
 * since bars scroll beneath it. The sheet places the row; the row fills it.
 */
export function TimelineRow({
  height,
  label,
  lane,
  scale,
  selected = false,
  highlighted = false,
  onLabelClick,
  labelTitle,
  laneRef,
  dropState = null,
}: TimelineRowProps) {
  const labelClass = cn(
    "border-subtle sticky left-0 z-5 flex min-w-0 items-center gap-2 overflow-hidden border-r px-2.5 text-left",
    selected ? "bg-surface-selected" : highlighted ? "bg-surface-hover" : "bg-card",
  );

  return (
    <div
      className="border-subtle relative grid border-b"
      style={{ gridTemplateColumns: `${LABEL_WIDTH}px ${scale.width}px`, height }}
    >
      {onLabelClick ? (
        <button
          type="button"
          className={cn(labelClass, "ui-focus-ring hover:bg-surface-hover")}
          title={labelTitle}
          aria-pressed={selected}
          onClick={onLabelClick}
        >
          {label}
        </button>
      ) : (
        <div className={labelClass} title={labelTitle}>
          {label}
        </div>
      )}
      <div
        ref={laneRef}
        className={cn(
          "relative overflow-hidden",
          dropState === "ok" &&
            "bg-brand-subtle outline-brand outline-2 -outline-offset-2 outline-dashed",
          dropState === "refused" && "bg-danger-subtle",
        )}
      >
        {lane}
      </div>
    </div>
  );
}

/** A group's header row. The sheet pins the current one under the axis. */
export function TimelineGroupRow({
  scale,
  label,
  count,
  swatchClassName,
  tail,
}: {
  scale: TimelineScale;
  label: ReactNode;
  count: ReactNode;
  swatchClassName?: string;
  tail?: ReactNode;
}) {
  return (
    <div
      className="bg-sunken border-subtle grid border-b"
      style={{ gridTemplateColumns: `${LABEL_WIDTH}px ${scale.width}px`, height: GROUP_HEIGHT }}
    >
      <div className="bg-sunken border-subtle text-muted-foreground sticky left-0 z-2 flex min-w-0 items-center gap-1.5 overflow-hidden border-r px-2.5 text-xs font-medium whitespace-nowrap">
        {swatchClassName ? (
          <span aria-hidden className={cn("size-2 shrink-0 rounded-full", swatchClassName)} />
        ) : null}
        <span className="truncate">{label}</span>
        <span className="ml-auto font-mono tabular-nums">{count}</span>
      </div>
      {/* No overflow clip here: it would make the cell, not the sheet, the tail's sticky scroller. */}
      <div className="min-w-0">
        {tail ? (
          <div
            className="text-muted-foreground sticky flex h-full max-w-[calc(100cqw-var(--tl-label))] min-w-0 items-center gap-2 overflow-hidden px-2.5 text-xs whitespace-nowrap"
            style={{ left: LABEL_WIDTH }}
          >
            {tail}
          </div>
        ) : null}
      </div>
    </div>
  );
}
