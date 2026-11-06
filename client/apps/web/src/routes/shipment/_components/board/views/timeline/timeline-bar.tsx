import {
  barGeometry,
  TimelineBarFrame,
  TimelineDrivenFill,
  type BarDragHandle,
} from "@/components/timeline/timeline-bar-frame";
import { formatTimelineTime } from "@/components/timeline/timeline-format";
import type { TimelineScale } from "@/components/timeline/timeline-scale";
import { STAGE_META } from "@/lib/shipment-board/stage";
import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { useEffect } from "react";
import { useShipmentBoardStore } from "../../store";

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
 * One load on the axis: the shared bar tinted in its stage, deeper where it has been driven,
 * with its stop windows as strips along the bottom edge and any overrun past an appointment
 * hatched.
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
  const hovered = useShipmentBoardStore((s) => s.highlightId === load.id);
  const setHighlightId = useShipmentBoardStore((s) => s.setHighlightId);

  // A bar that goes away under the pointer gets no pointer leave: it lets go of its highlight.
  useEffect(
    () => () => {
      if (useShipmentBoardStore.getState().highlightId === load.id) setHighlightId(null);
    },
    [load.id, setHighlightId],
  );

  const geometry = barGeometry(load.start, load.end, scale);
  if (!geometry) return null;
  const { width, local } = geometry;

  const uncovered = load.coverage.kind === "uncovered";
  const nextStop = load.eta ? load.stops[load.eta.stopIndex] : null;
  const late = !!load.eta && load.eta.lateMinutes > 0 && !!nextStop;
  const overrunFrom = late && nextStop ? local(nextStop.windowEnd) : width;

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

  return (
    <TimelineBarFrame
      id={load.id}
      geometry={geometry}
      scale={scale}
      top={top}
      height={height}
      toneVar={uncovered ? "--warning" : STAGE_META[load.stage].colorVar}
      caption={text}
      captionInside={
        caption === "lane" ? (
          <>
            <span className="mr-1 font-mono font-semibold">{pro}</span>
            {route}
          </>
        ) : undefined
      }
      captionTone={tone}
      captionRoom={Math.min(width, overrunFrom)}
      nextLeft={nextLeft}
      open={uncovered}
      conflict={conflict}
      selected={selected}
      hovered={hovered}
      label={t("{0}, {1}", pro, route)}
      onSelect={() => onSelect(load.id)}
      drag={drag}
      hoverItem={load}
      onHoverChange={(entered) => {
        if (entered) setHighlightId(load.id);
        else if (useShipmentBoardStore.getState().highlightId === load.id) setHighlightId(null);
      }}
    >
      {load.driven === "all" ? (
        <span aria-hidden className="tl-bar-driven" style={{ width }} />
      ) : load.driven === "underway" ? (
        <TimelineDrivenFill start={load.start} end={load.end} scale={scale} geometry={geometry} />
      ) : null}
      {late && nextStop && load.eta ? (
        <span
          aria-hidden
          className="tl-bar-overrun"
          style={{
            left: overrunFrom,
            width: Math.max(local(load.eta.at) - overrunFrom, 2),
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
              left: local(stop.windowStart),
              width: Math.max(local(stop.windowEnd) - local(stop.windowStart), 4),
            }}
          />
        ),
      )}
      {load.eta && load.eta.at >= scale.start && load.eta.at <= scale.end ? (
        <span
          aria-hidden
          className="tl-bar-eta"
          data-late={late}
          style={{ left: Math.min(local(load.eta.at), width - 5) }}
        />
      ) : null}
    </TimelineBarFrame>
  );
}
