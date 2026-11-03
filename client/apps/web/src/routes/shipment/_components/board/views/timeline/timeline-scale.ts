import type { TimelineRange } from "@/lib/shipment-board/timeline";

/** The sticky label column's width. */
export const LABEL_WIDTH = 220;
/** The axis row's height; group headers stick just under it. */
export const AXIS_HEIGHT = 44;
/** A group header's height. */
export const GROUP_HEIGHT = 26;
/** A shipment row's height, and the bar centred in it. */
export const ROW_HEIGHT = 40;
export const BAR_HEIGHT = 20;
/* A driver or carrier lane: its bars stack in rows LANE_ROW_STEP apart inside
   LANE_PAD of padding, and an empty or one-row lane is as tall as a shipment row. */
export const LANE_PAD = 7;
export const LANE_ROW_STEP = 26;

/** A lane's height for `rows` stacked bars. */
export function laneHeight(rows: number): number {
  return Math.max(ROW_HEIGHT, LANE_PAD * 2 + rows * LANE_ROW_STEP - (LANE_ROW_STEP - BAR_HEIGHT));
}

const PX_PER_HOUR: Record<TimelineRange, number> = { day: 64, "3d": 24, week: 10 };
const TICK_HOURS: Record<TimelineRange, number> = { day: 2, "3d": 6, week: 12 };
/** Hours of each day shaded as night, so the working day stands out. */
export const NIGHT_BANDS: readonly (readonly [number, number])[] = [
  [0, 6],
  [20, 24],
];

export type TimelineScale = {
  range: TimelineRange;
  /** Day boundaries: the start of each day shown, then the end of the last. */
  days: readonly number[];
  start: number;
  end: number;
  width: number;
  tickHours: number;
  /** A moment's distance in pixels from the start of the range. */
  x: (unix: number) => number;
};

/**
 * The range laid out at its zoom. `minWidth` is the room beside the label
 * column: a range narrower than it stretches to fill it, so a wide or zoomed-out
 * window never shows empty sheet past the range's end.
 */
export function createTimelineScale(
  range: TimelineRange,
  days: readonly number[],
  minWidth = 0,
): TimelineScale {
  const start = days[0];
  const end = days[days.length - 1];
  const pxPerSecond = Math.max(PX_PER_HOUR[range] / 3_600, minWidth / (end - start));
  return {
    range,
    days,
    start,
    end,
    width: Math.round((end - start) * pxPerSecond),
    tickHours: TICK_HOURS[range],
    x: (unix) => (unix - start) * pxPerSecond,
  };
}
