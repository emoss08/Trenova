"use no memo";

import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime, formatUnixMonthDay, formatUnixWeekday } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, type ReactNode, type Ref } from "react";
import { AXIS_HEIGHT, LABEL_WIDTH, NIGHT_BANDS, type TimelineScale } from "./timeline-scale";

const HOUR = 3_600;
/** How far left of now the canvas opens, so the hours just gone stay in view. */
const NOW_LEAD_PX = 160;

type TimelineCanvasProps = {
  scale: TimelineScale;
  today: number;
  labelHeading: string;
  children: ReactNode;
};

/**
 * The scrolling sheet every row sits on: a sticky time axis, night hours and
 * day boundaries behind the rows, and the now line above everything. The
 * label column is sticky too, and the now line hides once it scrolls in
 * behind it rather than drawing across the names.
 *
 * `"use no memo"`: the now line is moved by the scroll handler directly.
 */
export function TimelineCanvas({ scale, today, labelHeading, children }: TimelineCanvasProps) {
  // The canvas reads the clock itself: a minute passing moves the now line and
  // redraws nothing else, since the rows arrive as the same `children`.
  const now = useNow("minute");
  const scrollRef = useRef<HTMLDivElement>(null);
  const nowRef = useRef<HTMLDivElement>(null);
  const nowVisible = now > scale.start && now < scale.end;

  const clipNow = () => {
    const line = nowRef.current;
    const sheet = scrollRef.current;
    if (!line || !sheet) return;
    line.style.visibility = scale.x(now) - sheet.scrollLeft < 0 ? "hidden" : "visible";
  };

  // Open on now when it is in view, otherwise at the start of the range.
  useEffect(() => {
    const sheet = scrollRef.current;
    if (!sheet) return;
    sheet.scrollLeft = nowVisible ? Math.max(0, scale.x(now) - NOW_LEAD_PX) : 0;
    clipNow();
    // Only a new range moves the view; the minute ticking over does not.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scale.start, scale.end]);

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

  return (
    <div
      ref={scrollRef}
      className="absolute inset-0 overflow-auto overscroll-contain"
      onScroll={clipNow}
    >
      <div className="relative min-h-full" style={{ width: LABEL_WIDTH + scale.width }}>
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
        {children}

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
      className="bg-card border-subtle sticky top-0 z-8 grid border-b"
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
 * since bars scroll beneath it. A row off screen skips layout and paint
 * (`content-visibility`), with its known height held in its place, so a range
 * of hundreds of loads scrolls at the cost of the rows in view.
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
      style={{
        gridTemplateColumns: `${LABEL_WIDTH}px ${scale.width}px`,
        height,
        contentVisibility: "auto",
        containIntrinsicSize: `auto ${height}px`,
      }}
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

/** A group's header row, stuck under the axis while its rows scroll by. */
export function TimelineGroupRow({
  scale,
  label,
  count,
  swatchClassName,
  tail,
}: {
  scale: TimelineScale;
  label: string;
  count: number;
  swatchClassName?: string;
  tail?: ReactNode;
}) {
  return (
    <div
      className="bg-sunken border-subtle sticky z-7 grid h-6.5 border-b"
      style={{ gridTemplateColumns: `${LABEL_WIDTH}px ${scale.width}px`, top: AXIS_HEIGHT }}
    >
      <div className="bg-sunken border-subtle text-muted-foreground sticky left-0 flex min-w-0 items-center gap-1.5 overflow-hidden border-r px-2.5 text-xs font-medium whitespace-nowrap">
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
            className="text-muted-foreground sticky flex h-full items-center gap-2 px-2.5 text-xs whitespace-nowrap"
            style={{ left: LABEL_WIDTH }}
          >
            {tail}
          </div>
        ) : null}
      </div>
    </div>
  );
}
