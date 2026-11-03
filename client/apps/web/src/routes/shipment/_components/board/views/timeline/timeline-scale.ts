import type { TimelineRange } from "@/lib/shipment-board/timeline";

/** The sticky label column's width. */
export const LABEL_WIDTH = 220;
/** The axis row's height; group headers stick just under it. */
export const AXIS_HEIGHT = 44;

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

export function createTimelineScale(range: TimelineRange, days: readonly number[]): TimelineScale {
  const start = days[0];
  const end = days[days.length - 1];
  const pxPerSecond = PX_PER_HOUR[range] / 3_600;
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
