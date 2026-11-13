/**
 * The network chart's layout, top to bottom: a strip for the "now" label, a lane for each
 * line, then the bars either side of their own baseline. Each band keeps to its own rows so
 * nothing is drawn over anything else.
 */
export const TIDE_HEIGHT = 178;
/** The baseline of the "now" label, in the strip above the lanes. */
export const NOW_LABEL_Y = 11;

/** The rows a line is drawn between: its scale's top value at `top`, zero at `bottom`. */
export type Lane = { top: number; bottom: number };

/**
 * Loads and drivers each have a lane and a scale of their own. Loads are counted a window at
 * a time and drivers are everyone free at once, so a fleet with drivers to spare has many
 * times more of one than the other, and on a shared scale the loads were a flat line along
 * the bottom with their range and their busy hours pressed out of sight.
 */
export const LOADS_LANE: Lane = { top: 24, bottom: 68 };
export const DRIVERS_LANE: Lane = { top: 86, bottom: 110 };
export const BARS_BASE = 140;
/** The tallest a bar grows from its baseline, leaving room for its number. */
export const BAR_MAX = 14;
const BAR_MIN = 2;
/**
 * The count a bar lane's scale never drops below, so one load with no truck nearby draws as
 * one load and not as a full-height alarm.
 */
const BAR_FLOOR = 4;
/** The width a "now" label needs beside its line before it has to sit on the other side. */
const NOW_LABEL_ROOM = 84;

export type BoardColumns = { width: number; label: number; value: number; count: number };

export function columnStep({ width, label, value, count }: BoardColumns): number {
  return Math.max(0, width - label - value) / Math.max(count, 1);
}

/**
 * The top of a lane's scale: the first round number at or above its highest value, so the
 * line never leaves its lane and the number beside it is one a person can read at a glance.
 * The steps are fine enough that the line uses most of its lane. A lane with nothing in it
 * still has a scale to draw on.
 */
export function laneTop(highest: number): number {
  const top = Math.max(highest, 1);
  const raw = top / 5;
  const magnitude = 10 ** Math.floor(Math.log10(raw));
  const step = Math.max(
    1,
    [1, 2, 5, 10].map((multiple) => multiple * magnitude).find((size) => size >= raw) ?? raw,
  );
  return Math.ceil(top / step) * step;
}

/** Where a count sits in its lane: zero on the lane's bottom row, `top` on its top row. */
export function laneScale(lane: Lane, top: number): (value: number) => number {
  return (value) =>
    lane.bottom - (Math.min(Math.max(value, 0), top) / top) * (lane.bottom - lane.top);
}

/**
 * How tall a bar is for a count, in a lane scaled to its own largest value: loads with no
 * truck nearby and idle drivers are different things, counted in ones and in hundreds, and on
 * one scale the loads would not show at all. A count that rounds to nothing draws nothing.
 */
export function barHeights(values: readonly number[]): (value: number) => number {
  const top = Math.max(BAR_FLOOR, ...values);
  return (value) => (value < 0.5 ? 0 : Math.max(BAR_MIN, (value / top) * BAR_MAX));
}

/** Which side of its line the "now" label sits on: after it, unless that would leave the plot. */
export function nowLabelAnchor(nowX: number, plotRight: number): "start" | "end" {
  return nowX + NOW_LABEL_ROOM > plotRight ? "end" : "start";
}
