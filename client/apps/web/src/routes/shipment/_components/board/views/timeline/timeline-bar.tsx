import { usePointerHover } from "@/components/pointer-hover-card";
import { STAGE_META } from "@/lib/shipment-board/stage";
import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, type CSSProperties, type HTMLAttributes, type Ref } from "react";
import { useShipmentBoardStore } from "../../store";
import { captionWidth, formatTimelineTime } from "./timeline-format";
import type { TimelineScale } from "./timeline-scale";

export type BarDragHandle = {
  ref: Ref<HTMLButtonElement>;
  props: HTMLAttributes<HTMLButtonElement>;
  dragging: boolean;
};

type TimelineBarProps = {
  load: TimelineLoad;
  scale: TimelineScale;
  top: number;
  height: number;
  /** "timing" leads with when the load lands; "lane" with which load it is. */
  caption: "timing" | "lane";
  /** Where the next bar in the same row starts, in pixels; bounds a caption drawn beside this one. */
  nextLeft?: number;
  conflict?: boolean;
  selected: boolean;
  onSelect: (id: string) => void;
  drag?: BarDragHandle;
};

/**
 * One load on the axis: a pill tinted in its stage, deeper where it has been
 * driven, with its stop windows as strips along the bottom edge and any
 * overrun past an appointment hatched. The caption sits inside when it fits,
 * beside the bar when there is room, and otherwise only in the hover.
 */
export function TimelineBar({
  load,
  scale,
  top,
  height,
  caption,
  nextLeft,
  conflict = false,
  selected,
  onSelect,
  drag,
}: TimelineBarProps) {
  const t = useT();
  const hover = usePointerHover<TimelineLoad>();
  const hovered = useShipmentBoardStore((s) => s.highlightId === load.id);
  const setHighlightId = useShipmentBoardStore((s) => s.setHighlightId);

  // A bar that goes away under the pointer gets no pointer leave: it lets go of
  // its card and its highlight itself.
  useEffect(
    () => () => {
      hover?.release(load.id);
      if (useShipmentBoardStore.getState().highlightId === load.id) setHighlightId(null);
    },
    [hover, load.id, setHighlightId],
  );

  if (load.end <= scale.start || load.start >= scale.end) return null;

  const uncovered = load.coverage.kind === "uncovered";
  const toneVar = uncovered ? "--warning" : STAGE_META[load.stage].colorVar;
  const left = scale.x(Math.max(load.start, scale.start));
  const width = Math.max(scale.x(Math.min(load.end, scale.end)) - left, 8);
  const clampLocal = (unix: number) =>
    scale.x(Math.min(Math.max(unix, scale.start), scale.end)) - left;

  const nextStop = load.eta ? load.stops[load.eta.stopIndex] : null;
  const late = !!load.eta && load.eta.lateMinutes > 0 && !!nextStop;
  const overrunFrom = late && nextStop ? clampLocal(nextStop.windowEnd) : width;

  const pro = load.shipment.proNumber ?? "—";
  const route = [load.stops[0].city, load.stops[load.stops.length - 1].city].join(" → ");
  let text: string;
  let tone: "muted" | "late" | "plain" = "muted";
  if (caption === "lane") {
    text = [pro, route].join(" ");
    tone = "plain";
  } else if (load.stage === "Delivered") {
    const last = load.stops[load.stops.length - 1];
    text = t("Delivered {0}", formatTimelineTime(last.arrivedAt ?? last.windowEnd, scale));
  } else if (late && load.eta) {
    text = t(
      "{0} late · ETA {1}",
      formatMinutesSpan(load.eta.lateMinutes),
      formatTimelineTime(load.eta.at, scale),
    );
    tone = "late";
  } else if (load.eta) {
    text = t("ETA {0}", formatTimelineTime(load.eta.at, scale));
  } else {
    text = t(
      "{0} to {1}",
      formatTimelineTime(load.stops[0].windowStart, scale),
      formatTimelineTime(load.stops[load.stops.length - 1].windowStart, scale),
    );
  }

  const room = Math.min(width, overrunFrom);
  const fitsInside = captionWidth(text) <= room;
  const besideRoom = (nextLeft ?? scale.width) - (left + width) - 6;
  const fitsBeside = !fitsInside && besideRoom >= Math.min(captionWidth(text), 90);
  const captionClass = cn(
    "pointer-events-none truncate text-xs leading-none whitespace-nowrap",
    tone === "late" && "text-danger font-mono font-semibold",
    tone === "muted" && "text-muted-foreground font-mono",
    tone === "plain" && "text-foreground",
  );
  const style = {
    ...drag?.props.style,
    "--tone": `var(${toneVar})`,
    left,
    width,
    top,
    height,
    opacity: drag?.dragging ? 0.4 : undefined,
  } as CSSProperties;

  return (
    <>
      <button
        ref={drag?.ref}
        type="button"
        {...drag?.props}
        className="tl-bar ui-focus-ring"
        data-hovered={hovered}
        data-selected={selected}
        data-uncovered={uncovered}
        data-conflict={conflict}
        data-clip-start={load.start < scale.start}
        data-clip-end={load.end > scale.end}
        data-draggable={!!drag}
        style={style}
        aria-label={t("{0}, {1}", pro, route)}
        aria-pressed={selected}
        onClick={() => onSelect(load.id)}
        onPointerEnter={(event) => {
          setHighlightId(load.id);
          if (!drag?.dragging)
            hover?.enter(load, event.currentTarget, event.clientX, event.clientY);
        }}
        onPointerMove={(event) => hover?.move(event.clientX, event.clientY)}
        onPointerLeave={() => {
          if (useShipmentBoardStore.getState().highlightId === load.id) setHighlightId(null);
          hover?.leave();
        }}
        onFocus={(event) => {
          // Keyboard focus opens the card; the focus a click brings does not.
          if (event.currentTarget.matches(":focus-visible"))
            hover?.enter(load, event.currentTarget);
        }}
        onBlur={() => hover?.leave()}
        onKeyDown={(event) => {
          // Escape closes the card, and Space picks the load up: no card rides a drag.
          if (event.key === "Escape" || event.key === " ") hover?.dismiss();
          drag?.props.onKeyDown?.(event);
        }}
        onPointerDown={(event) => {
          hover?.dismiss();
          drag?.props.onPointerDown?.(event);
        }}
      >
        {load.driven === "all" ? (
          <span aria-hidden className="tl-bar-driven" style={{ width }} />
        ) : load.driven === "underway" ? (
          <DrivenUntilNow load={load} scale={scale} left={left} width={width} />
        ) : null}
        {late && nextStop && load.eta ? (
          <span
            aria-hidden
            className="tl-bar-overrun"
            style={{
              left: overrunFrom,
              width: Math.max(clampLocal(load.eta.at) - overrunFrom, 2),
            }}
          />
        ) : null}
        {load.stops.map((stop) =>
          stop.windowEnd < scale.start || stop.windowStart > scale.end ? null : (
            <span
              key={stop.id}
              aria-hidden
              className="tl-bar-window"
              data-done={stop.done}
              style={{
                left: clampLocal(stop.windowStart),
                width: Math.max(clampLocal(stop.windowEnd) - clampLocal(stop.windowStart), 4),
              }}
            />
          ),
        )}
        {load.eta && load.eta.at >= scale.start && load.eta.at <= scale.end ? (
          <span
            aria-hidden
            className="tl-bar-eta"
            data-late={late}
            style={{ left: Math.min(clampLocal(load.eta.at), width - 5) }}
          />
        ) : null}
        {fitsInside ? (
          <span
            className={cn(captionClass, "absolute top-1/2 left-1.5 -translate-y-1/2")}
            style={{ maxWidth: room - 10 }}
          >
            {caption === "lane" ? (
              <>
                <span className="mr-1 font-mono font-semibold">{pro}</span>
                {route}
              </>
            ) : (
              text
            )}
          </span>
        ) : null}
      </button>
      {fitsBeside ? (
        <span
          aria-hidden
          className={cn(captionClass, "absolute -translate-y-1/2")}
          style={{ left: left + width + 6, top: top + height / 2, maxWidth: besideRoom }}
        >
          {text}
        </span>
      ) : null}
      {conflict ? (
        <span
          aria-hidden
          className="bg-danger border-card pointer-events-none absolute z-2 size-2 -translate-x-1/2 -translate-y-1/2 rounded-full border-2"
          style={{ left: left + width, top }}
        />
      ) : null}
    </>
  );
}

/** The driven part of a load under way: the only piece of a bar the clock moves. */
function DrivenUntilNow({
  load,
  scale,
  left,
  width,
}: {
  load: TimelineLoad;
  scale: TimelineScale;
  left: number;
  width: number;
}) {
  const now = useNow("minute");
  const until = Math.min(Math.max(now, load.start), load.end);
  if (until <= scale.start) return null;
  const driven = scale.x(Math.min(until, scale.end)) - left;
  return (
    <span
      aria-hidden
      className="tl-bar-driven"
      style={{ width: Math.max(0, Math.min(driven, width)) }}
    />
  );
}
