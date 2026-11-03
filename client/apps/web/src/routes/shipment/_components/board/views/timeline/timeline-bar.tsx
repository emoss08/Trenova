import { STAGE_META } from "@/lib/shipment-board/stage";
import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { useNow } from "@trenova/shared/hooks/use-now";
import {
  createTooltipHandle,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  createContext,
  useContext,
  useState,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
  type Ref,
} from "react";
import { useShipmentBoardStore } from "../../store";
import { captionWidth, formatTimelineTime } from "./timeline-format";
import type { TimelineScale } from "./timeline-scale";

type BarTooltipHandle = ReturnType<typeof createTooltipHandle<TimelineLoad>>;

const BarTooltipContext = createContext<BarTooltipHandle | null>(null);

/**
 * The one hover detail every bar opens: each bar is a trigger on this tooltip,
 * passing its load as the payload, so a timeline of hundreds of loads mounts a
 * single tooltip rather than one per bar.
 */
export function TimelineBarTooltip({
  scale,
  children,
}: {
  scale: TimelineScale;
  children: ReactNode;
}) {
  const [handle] = useState(() => createTooltipHandle<TimelineLoad>());
  return (
    <BarTooltipContext.Provider value={handle}>
      {children}
      <Tooltip handle={handle}>
        {({ payload }) =>
          payload ? (
            <TooltipContent side="top" className="max-w-72">
              <TimelineBarDetail load={payload} scale={scale} />
            </TooltipContent>
          ) : null
        }
      </Tooltip>
    </BarTooltipContext.Provider>
  );
}

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
  const tooltip = useContext(BarTooltipContext);
  const hovered = useShipmentBoardStore((s) => s.highlightId === load.id);
  const setHighlightId = useShipmentBoardStore((s) => s.setHighlightId);

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
      <TooltipTrigger
        handle={tooltip ?? undefined}
        payload={load}
        disabled={!tooltip || drag?.dragging}
        render={
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
            onMouseEnter={() => setHighlightId(load.id)}
            onMouseLeave={() => {
              if (useShipmentBoardStore.getState().highlightId === load.id) setHighlightId(null);
            }}
          />
        }
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
      </TooltipTrigger>
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

/** The hover, drawn on the inverted tooltip surface: every stop with its time, lateness, and who has it. */
function TimelineBarDetail({ load, scale }: { load: TimelineLoad; scale: TimelineScale }) {
  const t = useT();
  const coverage = load.coverage;

  return (
    <div className="flex flex-col gap-1.5 text-xs">
      <div className="flex items-center gap-2">
        <span className="font-mono font-semibold">{load.shipment.proNumber ?? "—"}</span>
        <span className="truncate opacity-75">{load.shipment.customer?.name}</span>
      </div>
      <ol className="flex flex-col gap-0.5">
        {load.stops.map((stop, index) => {
          let when: string;
          if (stop.done) {
            when = stop.departedAt
              ? t("Left {0}", formatTimelineTime(stop.departedAt, scale))
              : t("Arrived {0}", formatTimelineTime(stop.arrivedAt ?? stop.windowStart, scale));
          } else if (load.eta && load.eta.stopIndex === index) {
            when = t("ETA {0}", formatTimelineTime(load.eta.at, scale));
          } else {
            when = formatTimelineTime(stop.windowStart, scale);
          }
          return (
            <li key={stop.id} className="grid grid-cols-[1rem_1fr_auto] gap-1.5">
              <span className="font-mono opacity-75">{index + 1}</span>
              <span className="truncate">{stop.place}</span>
              <span className="font-mono tabular-nums opacity-75">{when}</span>
            </li>
          );
        })}
      </ol>
      {load.eta && load.eta.lateMinutes > 0 ? (
        <span className="font-semibold">
          {t("{0} past the appointment window", formatMinutesSpan(load.eta.lateMinutes))}
        </span>
      ) : null}
      <span className="opacity-75">
        {coverage.kind === "driver" || coverage.kind === "carrier"
          ? coverage.name
          : coverage.kind === "tendered"
            ? t("Tendered, waiting on the carrier")
            : t("Not covered")}
      </span>
    </div>
  );
}
