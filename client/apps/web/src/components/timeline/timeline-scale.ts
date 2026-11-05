/** The sticky label column's width. */
export const LABEL_WIDTH = 220;
/** The axis row's height; group headers stick just under it. */
export const AXIS_HEIGHT = 44;
/** A group header's height. */
export const GROUP_HEIGHT = 26;
/** A one-bar row's height, and the bar centred in it. */
export const ROW_HEIGHT = 40;
export const BAR_HEIGHT = 20;
/* A lane: its bars stack in rows LANE_ROW_STEP apart inside LANE_PAD of padding, and an
   empty or one-row lane is as tall as a one-bar row. */
export const LANE_PAD = 7;
export const LANE_ROW_STEP = 26;

/** A lane's height for `rows` stacked bars. */
export function laneHeight(rows: number): number {
  return Math.max(ROW_HEIGHT, LANE_PAD * 2 + rows * LANE_ROW_STEP - (LANE_ROW_STEP - BAR_HEIGHT));
}

/** Where a bar in row `row` of a lane sits from the lane's top. */
export function laneBarTop(row: number): number {
  return LANE_PAD + row * LANE_ROW_STEP;
}

/** Hours of each day shaded as night, so the working day stands out. */
export const NIGHT_BANDS: readonly (readonly [number, number])[] = [
  [0, 6],
  [20, 24],
];

/**
 * How closely a timeline is drawn: pixels per hour at its narrowest, the hours between
 * gridlines, and whether those gridlines carry a clock label. Each page names its own zooms.
 */
export type TimelineZoom = {
  pxPerHour: number;
  tickHours: number;
  hourLabels: boolean;
};

export type TimelineScale = {
  /** Day boundaries: the start of each day shown, then the end of the last. */
  days: readonly number[];
  start: number;
  end: number;
  width: number;
  tickHours: number;
  hourLabels: boolean;
  /** One day shown: a time inside it needs no weekday to say which day it is. */
  singleDay: boolean;
  /** A moment's distance in pixels from the start of the range. */
  x: (unix: number) => number;
};

/**
 * The days laid out at a zoom. `minWidth` is the room beside the label column: a range
 * narrower than it stretches to fill it, so a wide or zoomed-out window never shows empty
 * sheet past the range's end.
 */
export function createTimelineScale(
  days: readonly number[],
  zoom: TimelineZoom,
  minWidth = 0,
): TimelineScale {
  const start = days[0];
  const end = days[days.length - 1];
  const pxPerSecond = Math.max(zoom.pxPerHour / 3_600, minWidth / (end - start));
  return {
    days,
    start,
    end,
    width: Math.round((end - start) * pxPerSecond),
    tickHours: zoom.tickHours,
    hourLabels: zoom.hourLabels,
    singleDay: days.length === 2,
    x: (unix) => (unix - start) * pxPerSecond,
  };
}
