import { boardDuration, boardHours } from "../board-card";
import { cardDay } from "../../../record-card/record-card-values";
import type { DeskBoardChartTick } from "./metric-types";

/** The width a chart is laid out at before its box is measured: the conversation's exhibit. */
export const CHART_WIDTH = 682;

/**
 * The height each chart form draws at, whatever its width, so the sketch that holds its place
 * while it is read reserves exactly what lands. Charts are drawn at the measured pixel width of
 * their exhibit, never scaled, so every text keeps the size its class gives it.
 */
export const CHART_HEIGHTS = {
  trend: 156,
  bars: 166,
  columns: 134,
  dots: 106,
} as const;

/** A ranked bar's pitch: a 14px bar and the gap under it. */
export const RANKED_PITCH = 26;
/** A spans row's pitch: the cover line, its bound mark and the room under them. */
export const SPANS_PITCH = 34;

/** The height of ranked bars: one pitch a row, less the last row's gap. */
export function rankedHeight(rows: number): number {
  return Math.max(1, rows) * RANKED_PITCH - 6;
}

/** The height of a spans chart: one pitch a row and the axis under them. */
export function spansHeight(rows: number): number {
  return Math.max(1, rows) * SPANS_PITCH + 30;
}

/** A decimal the server sent as a string, or null when it is absent or not a number. */
export function toNumber(value: string | null | undefined): number | null {
  if (value === null || value === undefined || value.trim() === "") return null;
  const number = Number(value);
  return Number.isFinite(number) ? number : null;
}

/** The minus sign set in figures, so a negative lines up with the digits beside it. */
const MINUS = "−";

function moneyDigits(magnitude: number): { min: number; max: number } {
  if (magnitude >= 1000) return { min: 0, max: 0 };
  if (magnitude >= 1) return { min: 2, max: 2 };
  return { min: 2, max: 3 };
}

/**
 * A value on a chart, in the chart's unit: money in its currency (whole above a thousand, to
 * the tenth of a cent below a dollar), a percent from its fraction, a duration from seconds,
 * a count whole.
 */
export function chartValue(value: number, unit: string, currency: string): string {
  const magnitude = Math.abs(value);
  switch (unit) {
    case "money": {
      const digits = moneyDigits(magnitude);
      return new Intl.NumberFormat(undefined, {
        style: "currency",
        currency: currency || "USD",
        minimumFractionDigits: digits.min,
        maximumFractionDigits: digits.max,
      }).format(value);
    }
    case "percent":
      return new Intl.NumberFormat(undefined, {
        style: "percent",
        maximumFractionDigits: 1,
      }).format(value);
    case "duration":
    case "hours": {
      const text = unit === "hours" ? boardHours(magnitude) : boardDuration(magnitude);
      return value < 0 ? `${MINUS}${text}` : text;
    }
    case "count":
      return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value);
    default:
      return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value);
  }
}

/** The digits a tick needs to tell it from the next, `step` apart. */
function stepDigits(step: number): number {
  if (step <= 0) return 0;
  return Math.min(3, Math.max(0, -Math.floor(Math.log10(step) + 1e-9)));
}

/** A tick's value on an axis whose ticks stand `step` apart: no more digits than the step needs. */
export function tickValue(value: number, step: number, unit: string, currency: string): string {
  switch (unit) {
    case "money": {
      const digits = stepDigits(step);
      const fraction = digits > 0 ? Math.max(2, digits) : 0;
      return new Intl.NumberFormat(undefined, {
        style: "currency",
        currency: currency || "USD",
        minimumFractionDigits: fraction,
        maximumFractionDigits: fraction,
      }).format(value);
    }
    case "percent":
      return new Intl.NumberFormat(undefined, {
        style: "percent",
        maximumFractionDigits: stepDigits(step * 100),
      }).format(value);
    default:
      return chartValue(value, unit, currency);
  }
}

/** A count of whole units with its sign: `+2`, `−4`, `0`. */
export function signedCount(value: number): string {
  const digits = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(
    Math.abs(value),
  );
  if (value > 0) return `+${digits}`;
  if (value < 0) return `${MINUS}${digits}`;
  return digits;
}

function niceStep(raw: number): number {
  if (!(raw > 0)) return 1;
  const power = 10 ** Math.floor(Math.log10(raw));
  const fraction = raw / power;
  const nice =
    fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 2.5 ? 2.5 : fraction <= 5 ? 5 : 10;
  return nice * power;
}

/** Few round ticks that hold every value from `low` to `high`, `count` of them or near it. */
export function niceTicks(low: number, high: number, count = 3): { ticks: number[]; step: number } {
  let min = Math.min(low, high);
  let max = Math.max(low, high);
  if (min === max) {
    const pad = min === 0 ? 1 : Math.abs(min) * 0.1;
    min -= pad;
    max += pad;
  }
  const step = niceStep((max - min) / Math.max(1, count - 1));
  const start = Math.floor(min / step + 1e-9) * step;
  const end = Math.ceil(max / step - 1e-9) * step;
  const ticks: number[] = [];
  for (let at = start; at <= end + step / 2; at += step) {
    ticks.push(Math.round(at / step) * step);
  }
  return { ticks, step };
}

/** A straight map from one interval onto another. */
export function linear(d0: number, d1: number, r0: number, r1: number): (value: number) => number {
  const span = d1 - d0;
  return (value) => (span === 0 ? (r0 + r1) / 2 : r0 + ((value - d0) / span) * (r1 - r0));
}

/** A point along the axis as written: its own label, or the day of its moment. */
export function tickText(tick: DeskBoardChartTick | undefined): string {
  if (!tick) return "";
  if (tick.label !== "") return tick.label;
  return tick.at ? cardDay(tick.at) : "";
}

/** The indexes of `count` points to label, no more than `most`, the first and as many as fit. */
export function sparseIndexes(count: number, most: number): number[] {
  if (count <= 0) return [];
  if (count <= most) return Array.from({ length: count }, (_, index) => index);
  const every = Math.ceil((count - 1) / Math.max(1, most - 1));
  return Array.from({ length: count }, (_, index) => index).filter((index) => index % every === 0);
}

/**
 * A name cut to the room it has in a chart, with an ellipsis; its whole text goes in the
 * mark's `<title>`. `perChar` is the width one character of its class takes, near enough.
 */
export function fitText(text: string, room: number, perChar: number): string {
  const fits = Math.max(1, Math.floor(room / perChar));
  if (text.length <= fits) return text;
  return `${text.slice(0, Math.max(1, fits - 1)).trimEnd()}…`;
}

/** The lanes a dots chart sets its dots in, from the middle out, inside the band. */
export const DOT_LANES = [47, 36, 58, 25, 69, 16] as const;

/**
 * Each dot's lane: the first from the middle out where no dot already stands within `gap`
 * pixels, so dots at the same value stack instead of hiding each other. A dot that finds no
 * free lane takes the lane whose nearest dot is furthest away, so a crowd spreads evenly and
 * never leaves the band.
 */
export function laneOf(
  xs: readonly number[],
  gap: number,
  lanes: readonly number[] = DOT_LANES,
): number[] {
  const taken: number[][] = lanes.map(() => []);
  return xs.map((x) => {
    const nearest = taken.map((dots) =>
      dots.reduce((least, other) => Math.min(least, Math.abs(other - x)), Infinity),
    );
    const free = nearest.findIndex((distance) => distance >= gap);
    const at =
      free !== -1
        ? free
        : nearest.reduce((best, distance, index) => (distance > nearest[best] ? index : best), 0);
    taken[at].push(x);
    return lanes[at];
  });
}

/** An id an SVG `url(#…)` can name: React's ids carry characters a fragment does not take. */
export function svgId(id: string): string {
  return `dk${id.replace(/[^a-zA-Z0-9_-]/g, "")}`;
}
