import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { ChartFrame } from "./chart-frame";
import {
  CHART_HEIGHTS,
  chartValue,
  laneOf,
  linear,
  niceTicks,
  tickValue,
  toNumber,
} from "./chart-values";
import { circleBox, placeLabels, spotsAround, thinTicks, type Box } from "./chart-labels";
import type { DeskBoardChart } from "./metric-types";

const H = CHART_HEIGHTS.dots;
const AXIS = 86;
const BAND_TOP = 12;
const BAND_BOTTOM = 78;
/** Dots nearer than this stand in different lanes. */
const DOT_GAP = 10;

type Dot = { value: number; marked: boolean; ghost: boolean; label: string };

/** Every unit's dot: a row's own dots by value, or the row's value when it has none. */
export function dotsOf(chart: DeskBoardChart): Dot[] {
  return chart.rows.flatMap((row) => {
    const own = row.dots.flatMap((dot) => {
      const value = toNumber(dot.value);
      if (value === null) return [];
      const marked = dot.marked || row.marked;
      return [{ value, marked, ghost: dot.ghost, label: dot.label || (marked ? row.label : "") }];
    });
    if (own.length > 0) return own;
    const value = row.value ? toNumber(row.value.number) : null;
    return value === null
      ? []
      : [{ value, marked: row.marked, ghost: false, label: row.marked ? row.label : "" }];
  });
}

/** Whether a dots chart has a dot to draw. */
export function dotsDraw(chart: DeskBoardChart): boolean {
  return dotsOf(chart).length > 0;
}

function lineText(chart: DeskBoardChart, t: TranslateFn): string {
  if (!chart.ref) return "";
  const value = chartValue(Number(chart.ref.value), chart.unit, chart.currency);
  return chart.ref.label === "" ? value : t("{0} {1}", chart.ref.label, value);
}

/** What the dots show and their headline number: how many, the line, and the marked ones. */
export function dotsLabel(chart: DeskBoardChart, title: string, t: TranslateFn): string {
  const dots = dotsOf(chart);
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const marked = new Intl.ListFormat(undefined, { type: "unit" }).format(
    dots
      .filter((dot) => dot.marked)
      .map((dot) =>
        dot.label === "" ? format(dot.value) : t("{0} {1}", dot.label, format(dot.value)),
      ),
  );
  const line = lineText(chart, t);
  if (line === "") {
    return marked === ""
      ? t("{0}: {1, plural, one {# dot} other {# dots}}", title, dots.length)
      : t("{0}: {1, plural, one {# dot} other {# dots}}; marked {2}", title, dots.length, marked);
  }
  return marked === ""
    ? t("{0}: {1, plural, one {# dot} other {# dots}} against {2}", title, dots.length, line)
    : t(
        "{0}: {1, plural, one {# dot} other {# dots}} against {2}; marked {3}",
        title,
        dots.length,
        line,
        marked,
      );
}

/** The most marked units named on a dots chart: the furthest from the line first. */
const LABEL_BUDGET = 4;
const DOT_R = 4.5;
const MARK_R = 5.5;

function DotsMarks({ chart, width, t }: { chart: DeskBoardChart; width: number; t: TranslateFn }) {
  const dots = dotsOf(chart);
  const low = toNumber(chart.band?.low);
  const high = toNumber(chart.band?.high);
  const line = chart.ref ? Number(chart.ref.value) : null;
  const all = dots
    .map((dot) => dot.value)
    .concat([low, high, line].filter((value): value is number => value !== null));
  const { ticks, step } = niceTicks(Math.min(...all), Math.max(...all), 4);
  const left = 40;
  const right = width - 42;
  const x = (value: number) =>
    Math.min(right, Math.max(left, linear(ticks[0], ticks.at(-1) ?? ticks[0], left, right)(value)));
  const placed = dots.map((dot, index) => ({ ...dot, index, cx: x(dot.value) }));
  const lanes = laneOf(
    placed.map((dot) => dot.cx),
    DOT_GAP,
  );
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const centre = line ?? (low !== null && high !== null ? (low + high) / 2 : null);
  const named = placed
    .filter((dot) => dot.marked)
    .sort((a, b) => (centre === null ? 0 : Math.abs(b.value - centre) - Math.abs(a.value - centre)))
    .slice(0, LABEL_BUDGET);
  const fleet = lineText(chart, t);
  const fleetAsk =
    line === null
      ? []
      : [
          {
            text: fleet,
            cls: "lb" as const,
            spots: [
              { x: x(line), y: 9, anchor: "middle" as const },
              { x: x(line) - 4, y: 9, anchor: "end" as const },
              { x: x(line) + 4, y: 9, anchor: "start" as const },
            ],
          },
        ];
  const obstacles: Box[] = placed.map((dot, index) =>
    circleBox(dot.cx, lanes[index], dot.marked ? MARK_R : DOT_R),
  );
  if (line !== null)
    obstacles.push({ x0: x(line) - 1, x1: x(line) + 1, y0: BAND_TOP, y1: BAND_BOTTOM });
  const asks = [
    ...fleetAsk,
    ...named.map((dot) => ({
      text: dot.label === "" ? format(dot.value) : dot.label,
      cls: "lb" as const,
      spots: spotsAround(dot.cx, lanes[dot.index], MARK_R),
    })),
  ];
  const spots = placeLabels(asks, obstacles, { x0: 0, x1: width, y0: 0, y1: AXIS - 1 });
  const fleetSpot = line === null ? null : spots[0];
  const labelSpots = spots.slice(fleetAsk.length);
  const axisTicks = thinTicks(
    ticks.map((tick) => ({
      x: x(tick),
      text: tickValue(tick, step, chart.unit, chart.currency),
      cls: "tk" as const,
    })),
    { x0: 0, x1: width },
  );

  return (
    <>
      {low !== null && high !== null && (
        <rect
          className="band q"
          x={x(Math.min(low, high))}
          y={BAND_TOP}
          width={Math.abs(x(high) - x(low))}
          height={BAND_BOTTOM - BAND_TOP}
        />
      )}
      {line !== null && (
        <line className="ln" x1={x(line)} y1={BAND_TOP} x2={x(line)} y2={BAND_BOTTOM} />
      )}
      {placed.map((dot, index) => (
        <circle
          key={dot.index}
          className={dot.ghost ? "dot gh" : dot.marked ? "dot w" : "dot q"}
          cx={dot.cx}
          cy={lanes[index]}
          r={dot.marked ? MARK_R : DOT_R}
        />
      ))}
      {fleetSpot && (
        <text className="lb" x={fleetSpot.x} y={fleetSpot.y} textAnchor={fleetSpot.anchor}>
          {fleet}
        </text>
      )}
      {named.map((dot, index) => {
        const spot = labelSpots[index];
        if (!spot) return null;
        return (
          <text
            key={dot.index}
            className={dot.ghost ? "lb" : "lb w"}
            x={spot.x}
            y={spot.y}
            textAnchor={spot.anchor}
          >
            {asks[fleetAsk.length + index].text}
          </text>
        );
      })}
      <line className="ax" x1={left} y1={AXIS + 0.5} x2={width - 10} y2={AXIS + 0.5} />
      {axisTicks.map(({ index, spot }) => (
        <text key={ticks[index]} className="tk" x={spot.x} y={H - 4} textAnchor={spot.anchor}>
          {tickValue(ticks[index], step, chart.unit, chart.currency)}
        </text>
      ))}
    </>
  );
}

/**
 * Units against a band around the fleet's line: a dot each, stacked in lanes inside the band
 * where they meet, the marked ones warn, the ghosts hollow. At most four marked units are
 * named, the furthest from the line first, each where it collides with no dot and no other
 * words; the rest stay unnamed. The value axis carries as many mono ticks as fit.
 */
export function DotsChart({ chart, title }: { chart: DeskBoardChart; title: string }) {
  const t = useT();
  return (
    <ChartFrame height={H} label={dotsLabel(chart, title, t)}>
      {(width) => <DotsMarks chart={chart} width={width} t={t} />}
    </ChartFrame>
  );
}
