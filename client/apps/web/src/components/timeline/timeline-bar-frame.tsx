import { usePointerHover } from "@/components/pointer-hover-card";
import { useNow } from "@trenova/shared/hooks/use-now";
import { cn } from "@trenova/shared/lib/utils";
import {
  useEffect,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
  type Ref,
} from "react";
import { captionWidth } from "./timeline-format";
import type { TimelineScale } from "./timeline-scale";

/** What a draggable bar hands the frame: dnd-kit's node ref, listeners and drag state. */
export type BarDragHandle = {
  ref: Ref<HTMLButtonElement>;
  props: HTMLAttributes<HTMLButtonElement>;
  dragging: boolean;
};

export type BarGeometry = {
  left: number;
  width: number;
  /** The span runs past the range's start or end, so that edge is drawn cut. */
  clipStart: boolean;
  clipEnd: boolean;
  /** A moment's distance from the bar's left edge, held inside the range. */
  local: (unix: number) => number;
};

/** Where a span sits on the scale, or null when none of it is in the range. */
export function barGeometry(start: number, end: number, scale: TimelineScale): BarGeometry | null {
  if (end <= scale.start || start >= scale.end) return null;
  const left = scale.x(Math.max(start, scale.start));
  const width = Math.max(scale.x(Math.min(end, scale.end)) - left, 8);
  return {
    left,
    width,
    clipStart: start < scale.start,
    clipEnd: end > scale.end,
    local: (unix) => scale.x(Math.min(Math.max(unix, scale.start), scale.end)) - left,
  };
}

type TimelineBarFrameProps = {
  id: string;
  geometry: BarGeometry;
  scale: TimelineScale;
  top: number;
  height: number;
  /** The colour token the bar is tinted in, such as `--warning`. */
  toneVar: string;
  /** The caption, measured to decide whether it fits inside, beside, or only in the hover. */
  caption: string;
  /** Drawn inside the bar in place of `caption`, when the caption fits there. */
  captionInside?: ReactNode;
  captionTone: "muted" | "late" | "plain";
  /** The room the caption may use inside the bar; the bar's width unless something covers it. */
  captionRoom?: number;
  /** Where the next bar in the same row starts, in pixels; bounds a caption drawn beside. */
  nextLeft?: number;
  /** A dashed outline: work nobody has yet. */
  open?: boolean;
  conflict?: boolean;
  selected: boolean;
  /** Lit from elsewhere, such as the row it belongs to being hovered. */
  hovered?: boolean;
  label: string;
  onSelect: () => void;
  drag?: BarDragHandle;
  /** What the timeline's hover card shows for this bar. */
  hoverItem?: unknown;
  onHoverChange?: (hovered: boolean) => void;
  children?: ReactNode;
};

/**
 * One bar on a timeline: a pill tinted in its tone (`timeline.css`), its caption inside when
 * it fits, beside it when there is room, and otherwise only in the hover card, a conflict dot
 * on its end, and the hover, focus and drag wiring every bar shares. What the bar draws
 * inside (driven time, stop windows, an ETA) is its page's, passed as children.
 */
export function TimelineBarFrame({
  id,
  geometry,
  scale,
  top,
  height,
  toneVar,
  caption,
  captionInside,
  captionTone,
  captionRoom,
  nextLeft,
  open = false,
  conflict = false,
  selected,
  hovered = false,
  label,
  onSelect,
  drag,
  hoverItem,
  onHoverChange,
  children,
}: TimelineBarFrameProps) {
  const hover = usePointerHover<unknown>();

  // A bar that goes away under the pointer gets no pointer leave: it lets go of its card itself.
  useEffect(() => () => hover?.release(id), [hover, id]);

  const { left, width } = geometry;
  const room = captionRoom ?? width;
  const fitsInside = captionWidth(caption) <= room;
  const besideRoom = (nextLeft ?? scale.width) - (left + width) - 6;
  const fitsBeside = !fitsInside && besideRoom >= Math.min(captionWidth(caption), 90);
  const captionClass = cn(
    "pointer-events-none truncate text-xs leading-none whitespace-nowrap",
    captionTone === "late" && "text-danger font-mono font-semibold",
    captionTone === "muted" && "text-muted-foreground font-mono",
    captionTone === "plain" && "text-foreground",
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
        data-open={open}
        data-conflict={conflict}
        data-clip-start={geometry.clipStart}
        data-clip-end={geometry.clipEnd}
        data-draggable={!!drag}
        style={style}
        aria-label={label}
        aria-pressed={selected}
        onClick={onSelect}
        onPointerEnter={(event) => {
          onHoverChange?.(true);
          if (hoverItem !== undefined && !drag?.dragging)
            hover?.enter(hoverItem, event.currentTarget, event.clientX, event.clientY);
        }}
        onPointerMove={(event) => hover?.move(event.clientX, event.clientY)}
        onPointerLeave={() => {
          onHoverChange?.(false);
          hover?.leave();
        }}
        onFocus={(event) => {
          // Keyboard focus opens the card; the focus a click brings does not.
          if (hoverItem !== undefined && event.currentTarget.matches(":focus-visible"))
            hover?.enter(hoverItem, event.currentTarget);
        }}
        onBlur={() => hover?.leave()}
        onKeyDown={(event) => {
          // Escape closes the card, and Space picks the bar up: no card rides a drag.
          if (event.key === "Escape" || event.key === " ") hover?.dismiss();
          drag?.props.onKeyDown?.(event);
        }}
        onPointerDown={(event) => {
          hover?.dismiss();
          drag?.props.onPointerDown?.(event);
        }}
      >
        {children}
        {fitsInside ? (
          <span
            className={cn(captionClass, "absolute top-1/2 left-1.5 -translate-y-1/2")}
            style={{ maxWidth: room - 10 }}
          >
            {captionInside ?? caption}
          </span>
        ) : null}
      </button>
      {fitsBeside ? (
        <span
          aria-hidden
          className={cn(captionClass, "absolute -translate-y-1/2")}
          style={{ left: left + width + 6, top: top + height / 2, maxWidth: besideRoom }}
        >
          {caption}
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

/**
 * The driven part of a bar under way: deeper tint from its start up to now. It is the only
 * piece of a bar the clock moves, so a minute passing redraws only this.
 */
export function TimelineDrivenFill({
  start,
  end,
  scale,
  geometry,
}: {
  start: number;
  end: number;
  scale: TimelineScale;
  geometry: BarGeometry;
}) {
  const now = useNow("minute");
  const until = Math.min(Math.max(now, start), end);
  if (until <= scale.start) return null;
  const driven = scale.x(Math.min(until, scale.end)) - geometry.left;
  return (
    <span
      aria-hidden
      className="tl-bar-driven"
      style={{ width: Math.max(0, Math.min(driven, geometry.width)) }}
    />
  );
}
