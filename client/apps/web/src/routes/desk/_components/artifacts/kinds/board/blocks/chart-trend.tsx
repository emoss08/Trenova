import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { ChartFrame } from "./chart-frame";
import {
  CHART_HEIGHTS,
  chartValue,
  linear,
  niceTicks,
  tickText,
  tickValue,
  toNumber,
} from "./chart-values";
import {
  circleBox,
  lineBoxes,
  placeLabels,
  spotsAround,
  thinTicks,
  type Box,
  type Spot,
} from "./chart-labels";
import type { DeskBoardChart, DeskBoardChartLine } from "./metric-types";

const H = CHART_HEIGHTS.trend;
const AXIS = 130;
const TOP = 20;
const TICK_Y = H - 7;
/** Above this many points the line carries no dot per point, only the latest. */
const DOTTED = 12;
/** One character of the mono tick face, near enough to leave the axis labels room. */
const TICK_CHAR = 6.4;

type Point = { index: number; value: number };

/** The series a trend draws dark: the one marked now, else the last. */
export function nowSeries(chart: DeskBoardChart) {
  return chart.series.find((series) => series.role === "now") ?? chart.series.at(-1) ?? null;
}

function pointsOf(values: readonly (string | null)[]): (number | null)[] {
  return values.map((value) => toNumber(value));
}

/** Whether a trend has a point to draw. */
export function trendDraws(chart: DeskBoardChart): boolean {
  const series = nowSeries(chart);
  return series !== null && pointsOf(series.points).some((value) => value !== null);
}

/** Runs of consecutive points, broken where a point is missing. */
function runs(values: readonly (number | null)[], until: number): Point[][] {
  const out: Point[][] = [];
  let run: Point[] = [];
  for (let index = 0; index < until; index += 1) {
    const value = values[index];
    if (value === null || value === undefined) {
      if (run.length > 0) out.push(run);
      run = [];
      continue;
    }
    run.push({ index, value });
  }
  if (run.length > 0) out.push(run);
  return out;
}

function lineText(line: DeskBoardChartLine, chart: DeskBoardChart, t: TranslateFn): string {
  const value = chartValue(Number(line.value), chart.unit, chart.currency);
  return line.label === "" ? value : t("{0} {1}", line.label, value);
}

/** What the trend shows and its headline number, for the chart's label. */
export function trendLabel(chart: DeskBoardChart, title: string, t: TranslateFn): string {
  const series = nowSeries(chart);
  const values = series ? pointsOf(series.points) : [];
  const ghostAt = chart.ghost ? values.length - 1 : -1;
  const drawn = values
    .map((value, index) => ({ value, index }))
    .filter((point): point is Point => point.value !== null && point.index !== ghostAt);
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const latest = drawn.at(-1);
  if (!latest) {
    const ghost = values[ghostAt];
    return ghost === null || ghost === undefined
      ? title
      : t("{0}: {1} so far, still running", title, format(ghost));
  }
  if (drawn.length === 1) return t("{0}: {1}", title, format(latest.value));
  const lowest = drawn.reduce((low, point) => (point.value < low.value ? point : low));
  const first = tickText(chart.ticks[drawn[0].index]);
  const last = tickText(chart.ticks[latest.index]);
  const lowestAt = tickText(chart.ticks[lowest.index]);
  const ghost = ghostAt >= 0 ? values[ghostAt] : null;
  if (ghost !== null && ghost !== undefined) {
    return t(
      "{0}, {1} to {2}: latest {3}; lowest {4}, {5}; {6} so far in the period still running",
      title,
      first,
      last,
      format(latest.value),
      format(lowest.value),
      lowestAt,
      format(ghost),
    );
  }
  return t(
    "{0}, {1} to {2}: latest {3}; lowest {4}, {5}",
    title,
    first,
    last,
    format(latest.value),
    format(lowest.value),
    lowestAt,
  );
}

function TrendMarks({ chart, width, t }: { chart: DeskBoardChart; width: number; t: TranslateFn }) {
  const now = nowSeries(chart);
  if (!now) return null;
  const values = pointsOf(now.points);
  const was = chart.series
    .filter((series) => series !== now)
    .map((series) => pointsOf(series.points));
  const count = Math.max(values.length, chart.ticks.length);
  const ghostAt = chart.ghost && values.at(-1) !== null ? values.length - 1 : -1;
  const marked = new Set(chart.marked);

  const lineValues = [chart.target, chart.ref].flatMap((line) =>
    line ? [Number(line.value)] : [],
  );
  const bandValues = [toNumber(chart.band?.low), toNumber(chart.band?.high)].filter(
    (value): value is number => value !== null,
  );
  const all = [...values, ...was.flat()]
    .filter((value): value is number => value !== null)
    .concat(lineValues, bandValues);
  const { ticks, step } = niceTicks(Math.min(...all), Math.max(...all), 3);
  const top = Math.max(ticks.at(-1) ?? 0, ...all);
  const y = linear(ticks[0], top, AXIS, TOP);
  const tickLabels = ticks.map((tick) => tickValue(tick, step, chart.unit, chart.currency));
  const left = Math.max(40, Math.max(...tickLabels.map((label) => label.length)) * TICK_CHAR + 8);
  const right = width - 10;
  const last = width - 32;
  const pitch = count > 1 ? (last - left) / (count - 1) : 76;
  const x = (index: number) => (count > 1 ? left + index * pitch : (left + last) / 2);
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);

  const solid = runs(values, ghostAt >= 0 ? ghostAt : values.length);
  const drawn = solid.flat();
  const latest = drawn.at(-1);
  const ghostFrom = ghostAt >= 0 ? drawn.at(-1) : undefined;
  const band = chart.band;
  const bandFrom = band?.from ?? null;
  const bandTo = band?.to ?? bandFrom;

  const lines = (
    [
      ["tgt", chart.target],
      ["ref", chart.ref],
    ] as const
  ).flatMap(([kind, line]) =>
    line ? [{ kind, at: y(Number(line.value)), text: lineText(line, chart, t) }] : [],
  );
  const dotted = drawn.flatMap((point) => {
    const isMarked = marked.has(point.index);
    const isLatest = point === latest;
    if (!isMarked && !isLatest && count > DOTTED) return [];
    return [
      {
        index: point.index,
        cx: x(point.index),
        cy: y(point.value),
        r: isMarked ? 5.5 : isLatest ? 4.5 : 4,
        cls: isMarked ? "dot w" : isLatest ? "dot" : "dot q",
      },
    ];
  });
  const onChart = (run: Point[]) => run.map((point) => ({ x: x(point.index), y: y(point.value) }));
  const obstacles: Box[] = [
    ...solid.flatMap((run) => lineBoxes(onChart(run))),
    ...was.flatMap((series) =>
      runs(series, series.length).flatMap((run) => lineBoxes(onChart(run))),
    ),
    ...(ghostAt >= 0 && ghostFrom
      ? lineBoxes([
          { x: x(ghostFrom.index), y: y(ghostFrom.value) },
          { x: x(ghostAt), y: y(values[ghostAt] ?? 0) },
        ])
      : []),
    ...dotted.map((dot) => circleBox(dot.cx, dot.cy, dot.r)),
    ...lines.map((line) => ({ x0: left, x1: right, y0: line.at - 0.5, y1: line.at + 0.5 })),
  ];
  const labels: { text: string; className: string; spots: Spot[]; cls: "lb" | "tk" }[] = [];
  if (bandFrom !== null && bandTo !== null) {
    labels.push({
      text:
        bandFrom === bandTo
          ? tickText(chart.ticks[bandFrom])
          : t("{0} to {1}", tickText(chart.ticks[bandFrom]), tickText(chart.ticks[bandTo])),
      className: "lb w",
      cls: "lb",
      spots: [
        { x: (x(bandFrom) + x(bandTo)) / 2, y: 12, anchor: "middle" },
        { x: x(bandFrom) - pitch / 2, y: 12, anchor: "start" },
        { x: x(bandTo) + pitch / 2, y: 12, anchor: "end" },
      ],
    });
  }
  for (const point of drawn.filter((entry) => marked.has(entry.index))) {
    const cx = x(point.index);
    const cy = y(point.value);
    labels.push({
      text: format(point.value),
      className: "lb w",
      cls: "lb",
      spots: [spotsAround(cx, cy, 5.5)[1], ...spotsAround(cx, cy, 5.5).filter((_, at) => at !== 1)],
    });
  }
  if (latest && !marked.has(latest.index)) {
    const around = spotsAround(x(latest.index), y(latest.value), 4.5);
    labels.push({
      text: format(latest.value),
      className: "lb",
      cls: "lb",
      spots: count > DOTTED ? [around[2], around[0], around[1], around[3]] : around,
    });
  }
  for (const line of lines) {
    labels.push({
      text: line.text,
      className: "tk",
      cls: "tk",
      spots: [
        { x: right, y: line.at + 14, anchor: "end" },
        { x: right, y: line.at - 5, anchor: "end" },
        { x: left + 2, y: line.at + 14, anchor: "start" },
        { x: left + 2, y: line.at - 5, anchor: "start" },
      ],
    });
  }
  const spots = placeLabels(labels, obstacles, { x0: left, x1: width, y0: 0, y1: AXIS - 1 });
  const axisTicks = thinTicks(
    Array.from({ length: count }, (_, index) => ({
      x: x(index),
      text: tickText(chart.ticks[index]),
      cls: chart.ticks[index]?.marked ? ("lb" as const) : ("tk" as const),
      first: chart.ticks[index]?.marked,
    })),
    { x0: left - 20, x1: width },
  );

  return (
    <>
      {bandFrom !== null && bandTo !== null && (
        <rect
          className="band"
          x={x(bandFrom) - pitch / 2}
          y={0}
          width={x(bandTo) - x(bandFrom) + pitch}
          height={AXIS}
        />
      )}
      {bandFrom === null && bandValues.length === 2 && (
        <rect
          className="band q"
          x={left}
          y={y(Math.max(...bandValues))}
          width={right - left}
          height={Math.abs(y(bandValues[0]) - y(bandValues[1]))}
        />
      )}
      <line className="ax" x1={left} y1={AXIS + 0.5} x2={right} y2={AXIS + 0.5} />
      {lines.map((line) => (
        <line
          key={line.kind}
          className={line.kind}
          x1={left}
          y1={line.at}
          x2={right}
          y2={line.at}
        />
      ))}
      {ticks.map((tick, index) => (
        <text key={tick} className="tk" x={0} y={y(tick) + 4}>
          {tickLabels[index]}
        </text>
      ))}
      {was.map((series, index) =>
        runs(series, series.length).map((run) => (
          <polyline
            // oxlint-disable-next-line react/no-array-index-key -- series keep their order
            key={`${index}-${run[0].index}`}
            className="ln was"
            points={run.map((point) => `${x(point.index)},${y(point.value)}`).join(" ")}
          />
        )),
      )}
      {solid.map((run) => (
        <polyline
          key={run[0].index}
          className="ln"
          points={run.map((point) => `${x(point.index)},${y(point.value)}`).join(" ")}
        />
      ))}
      {ghostAt >= 0 && ghostFrom && (
        <line
          className="ln gh"
          x1={x(ghostFrom.index)}
          y1={y(ghostFrom.value)}
          x2={x(ghostAt)}
          y2={y(values[ghostAt] ?? 0)}
        />
      )}
      {dotted.map((dot) => (
        <circle key={dot.index} className={dot.cls} cx={dot.cx} cy={dot.cy} r={dot.r} />
      ))}
      {ghostAt >= 0 && (
        <circle className="dot gh" cx={x(ghostAt)} cy={y(values[ghostAt] ?? 0)} r={4} />
      )}
      {labels.map((label, index) => {
        const spot = spots[index];
        return spot ? (
          // oxlint-disable-next-line react/no-array-index-key -- labels keep their priority order
          <text
            key={index}
            className={label.className}
            x={spot.x}
            y={spot.y}
            textAnchor={spot.anchor}
          >
            {label.text}
          </text>
        ) : null;
      })}
      {axisTicks.map(({ index, spot }) => (
        <text
          key={`x${index}`}
          className={chart.ticks[index]?.marked ? "lb" : "tk"}
          x={spot.x}
          y={TICK_Y}
          textAnchor={spot.anchor}
        >
          {tickText(chart.ticks[index])}
        </text>
      ))}
    </>
  );
}

/**
 * A trend: one dark line of the series now, earlier series lighter, the marked points warn and
 * labelled, the latest labelled, a band over the marked stretch (or between two values), the
 * target and a reference as dashed lines, the last point a ghost while it still runs. Every
 * label is placed clear of the lines, the dots and the other labels, or not drawn: the band's
 * name first, then the marked points, the latest, the lines' names. Three mono ticks up the
 * side, as many along the foot as fit, no lattice.
 */
export function TrendChart({ chart, title }: { chart: DeskBoardChart; title: string }) {
  const t = useT();
  return (
    <ChartFrame height={H} label={trendLabel(chart, title, t)}>
      {(width) => <TrendMarks chart={chart} width={width} t={t} />}
    </ChartFrame>
  );
}
