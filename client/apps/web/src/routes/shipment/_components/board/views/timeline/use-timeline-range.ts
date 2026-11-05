import {
  createTimelineScale,
  LABEL_WIDTH,
  type TimelineZoom,
} from "@/components/timeline/timeline-scale";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import {
  dayStartOfIso,
  RANGE_DAYS,
  rangeFirstDay,
  type TimelineRange,
} from "@/lib/shipment-board/timeline";
import { dayBoundaries } from "@/lib/timeline/time-scale";
import { getTodayDate } from "@trenova/shared/lib/date";
import { useState, useSyncExternalStore } from "react";
import { useShipmentBoardUrl } from "../../url-state";

/** How closely each range is drawn: a week is too long to label every gridline's hour. */
const RANGE_ZOOM: Record<TimelineRange, TimelineZoom> = {
  day: { pxPerHour: 64, tickHours: 2, hourLabels: true },
  "3d": { pxPerHour: 24, tickHours: 6, hourLabels: true },
  week: { pxPerHour: 10, tickHours: 12, hourLabels: false },
};

/**
 * The width of the element `ref` is attached to, kept current as the window
 * resizes or zooms. It is read before paint, so the first frame is already laid
 * out at the real width.
 */
function useWidth() {
  const [node, setNode] = useState<HTMLElement | null>(null);
  const width = useSyncExternalStore(
    (notify) => {
      if (!node || typeof ResizeObserver === "undefined") return () => {};
      const observer = new ResizeObserver(notify);
      observer.observe(node);
      return () => observer.disconnect();
    },
    () => node?.clientWidth ?? 0,
  );
  return [setNode, width] as const;
}

/**
 * The range the URL asks for, laid out on the axis: the timeline draws it and
 * its skeleton draws the same axis while the range loads. Attach `sheetRef` to
 * the element the sheet spans: a range narrower than it stretches to fill it.
 */
export function useTimelineRange() {
  const [sheetRef, sheetWidth] = useWidth();
  const timezone = useUserTimezone();
  const [{ tl: range, tlday, tlrows: mode, qf }, setUrl] = useShipmentBoardUrl();
  const today = getTodayDate(timezone);
  const anchor = (tlday ? dayStartOfIso(tlday, timezone) : null) ?? today;
  const days = RANGE_DAYS[range];
  const scale = createTimelineScale(
    dayBoundaries(rangeFirstDay(range, anchor, timezone), days, timezone),
    RANGE_ZOOM[range],
    Math.max(0, sheetWidth - LABEL_WIDTH),
  );
  return { timezone, range, mode, qf, today, days, scale, setUrl, sheetRef };
}
