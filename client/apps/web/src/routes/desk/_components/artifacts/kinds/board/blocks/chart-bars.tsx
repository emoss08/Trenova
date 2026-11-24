import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatBoardValue } from "../board-card";
import { ChartFrame } from "./chart-frame";
import {
  CHART_HEIGHTS,
  chartValue,
  fitText,
  linear,
  rankedHeight,
  RANKED_PITCH,
  tickText,
  toNumber,
} from "./chart-values";
import {
  placeLabels,
  textWidth,
  thinTicks,
  type Box,
  type LabelAsk,
  type Spot,
} from "./chart-labels";
import type { DeskBoardChart, DeskBoardChartRow } from "./metric-types";

/** How a bars chart is drawn: ranked rows, period against period, or stacked periods. */
export type BarsLayout = "ranked" | "periods" | "stacked";

export function barsLayout(chart: DeskBoardChart): BarsLayout {
  if (chart.rows.length > 0) return "ranked";
  return chart.series.some((series) => series.role === "stack") ? "stacked" : "periods";
}

/** Whether a bars chart has a bar to draw. */
export function barsDraw(chart: DeskBoardChart): boolean {
  if (chart.rows.length > 0) return true;
  return chart.series.some((series) => series.points.some((point) => toNumber(point) !== null));
}

/** The height a bars chart draws at: one pitch a ranked row, else the plotted height. */
export function barsHeight(chart: DeskBoardChart): number {
  return barsLayout(chart) === "ranked" ? rankedHeight(chart.rows.length) : CHART_HEIGHTS.bars;
}

function rowNumber(row: DeskBoardChartRow): number | null {
  return row.value ? toNumber(row.value.number) : null;
}

/** A bar's corner: 2px, less on a bar too short to hold it. */
function radius(size: number): number {
  return Math.max(0, Math.min(2, size / 2));
}

// ------------------------------------------------------------------ ranked

/** The share of the width the row names may take before they are cut, and the least they get. */
const NAME_SHARE = 0.45;
const MIN_COLUMN = 80;
const MIN_VALUE_ROOM = 40;
const NAME_CHAR = 6.9;

function RankedMarks({
  chart,
  width,
  t,
}: {
  chart: DeskBoardChart;
  width: number;
  t: TranslateFn;
}) {
  const numbers = chart.rows.map(rowNumber);
  const known = numbers.filter((value): value is number => value !== null);
  const low = Math.min(0, ...known);
  const high = Math.max(0, ...known);
  const longest = Math.max(0, ...chart.rows.map((row) => textWidth(row.label, "nm")));
  const column = Math.round(Math.min(width * NAME_SHARE, Math.max(MIN_COLUMN, longest + 12)));
  const values = chart.rows.map((row, index) => {
    const value = numbers[index];
    if (value === null) return "";
    return row.value ? formatBoardValue(row.value) : chartValue(value, chart.unit, chart.currency);
  });
  const valueRoom = Math.max(MIN_VALUE_ROOM, ...values.map((text) => textWidth(text, "lb") + 14));
  const negativeRoom = low < 0 ? valueRoom : 0;
  const x = linear(low, high, column + negativeRoom, width - valueRoom);
  const zero = x(0);
  return (
    <>
      {chart.rows.map((row, index) => {
        const top = index * RANKED_PITCH;
        const value = numbers[index];
        const name = fitText(row.label, column - 10, NAME_CHAR);
        if (value === null) {
          return (
            // oxlint-disable-next-line react/no-array-index-key -- ranked rows keep their order
            <g key={index}>
              <text className="nm q" x={0} y={top + 12}>
                {name !== row.label && <title>{row.label}</title>}
                {name}
              </text>
              <text className="tk" x={column} y={top + 12}>
                {t("Not yours to read")}
              </text>
            </g>
          );
        }
        const end = x(value);
        const start = Math.min(zero, end);
        const size = Math.abs(end - zero);
        const text = values[index];
        return (
          // oxlint-disable-next-line react/no-array-index-key -- ranked rows keep their order
          <g key={index}>
            <text className={cn("nm", row.rest && "q")} x={0} y={top + 12}>
              {name !== row.label && <title>{row.label}</title>}
              {name}
            </text>
            <rect
              className={cn("bar", row.rest ? "was" : row.marked && "w")}
              x={start}
              y={top + 1}
              width={Math.max(size, value === 0 ? 0 : 2)}
              height={14}
              rx={radius(size)}
            />
            <text
              className={cn("lb", row.marked && "w")}
              x={value < 0 ? start - 8 : start + Math.max(size, 2) + 8}
              y={top + 12}
              textAnchor={value < 0 ? "end" : "start"}
            >
              {text}
            </text>
          </g>
        );
      })}
    </>
  );
}

// ----------------------------------------------------------- period on period

const AXIS = 140;
const PLOT_TOP = 30;
const TICK_Y = CHART_HEIGHTS.bars - 8;

function PeriodMarks({
  chart,
  width,
  t,
}: {
  chart: DeskBoardChart;
  width: number;
  t: TranslateFn;
}) {
  const series = chart.series.map((entry) => ({
    ...entry,
    values: entry.points.map((point) => toNumber(point)),
  }));
  const count = Math.max(chart.ticks.length, ...series.map((entry) => entry.values.length));
  const known = series
    .flatMap((entry) => entry.values)
    .filter((value): value is number => value !== null);
  const low = Math.min(0, ...known);
  const high = Math.max(0, ...known);
  const y = linear(low, high, AXIS, PLOT_TOP);
  const zero = y(0);
  const left = 40;
  const right = width - 10;
  const step = (right - left) / Math.max(1, count);
  const barWidth = Math.max(6, Math.min(28, (step - 16) / Math.max(1, series.length) - 4));
  const groupWidth = series.length * barWidth + (series.length - 1) * 4;
  const marked = new Set(chart.marked);
  const nowAt = series.findIndex((entry) => entry.role === "now");
  const dark = nowAt === -1 ? series.length - 1 : nowAt;
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const named = (at: number, value: number) =>
    series[at].name === "" ? format(value) : t("{0} {1}", series[at].name, format(value));
  const darkValues = series[dark]?.values ?? [];
  const largest = darkValues.reduce<number>(
    (best, value, index) =>
      value !== null && (best === -1 || value > (darkValues[best] ?? -Infinity)) ? index : best,
    -1,
  );

  const groups = Array.from({ length: count }, (_, index) => {
    const centre = left + step * (index + 0.5);
    const start = centre - groupWidth / 2;
    const bars = series.map((entry, at) => {
      const value = entry.values[index];
      if (value === null || value === undefined) return null;
      const top = Math.min(y(value), zero);
      const size = Math.abs(y(value) - zero);
      return { at, value, top, size, x: start + at * (barWidth + 4), was: at !== dark };
    });
    return { index, centre, marked: marked.has(index), bars };
  });
  const obstacles: Box[] = groups.flatMap((group) =>
    group.bars.flatMap((bar) =>
      bar ? [{ x0: bar.x, x1: bar.x + barWidth, y0: bar.top, y1: bar.top + bar.size }] : [],
    ),
  );
  type Ask = { text: string; cls: "lb" | "tk"; className: string; spots: Spot[] };
  const above = (bar: { x: number; top: number }, lift = 0): Spot[] => [
    { x: bar.x + barWidth / 2, y: bar.top - 7 - lift, anchor: "middle" },
    { x: bar.x + barWidth / 2, y: bar.top - 19 - lift, anchor: "middle" },
  ];
  const asks: Ask[] = [];
  const byPriority = [
    ...groups.filter((group) => group.marked),
    ...groups.filter((group) => !group.marked && group.index === largest),
    ...groups.filter((group) => !group.marked && group.index !== largest),
  ];
  for (const group of byPriority) {
    const own = group.bars[dark];
    if (!own) continue;
    const before = group.bars.find((bar) => bar !== null && bar.was);
    if (group.marked && before) {
      asks.push({
        text: named(dark, own.value),
        cls: "lb",
        className: "lb w",
        spots: [
          {
            x: own.x + barWidth / 2,
            y: Math.min(own.top - 7, before.top - 25),
            anchor: "middle",
          },
          ...above(own),
        ],
      });
      asks.push({
        text: named(before.at, before.value),
        cls: "tk",
        className: "tk",
        spots: [
          ...above(before),
          { x: before.x + barWidth, y: before.top - 7, anchor: "end" },
          { x: before.x + barWidth, y: before.top - 19, anchor: "end" },
        ],
      });
    } else {
      asks.push({
        text: format(own.value),
        cls: "lb",
        className: group.marked ? "lb w" : "lb",
        spots: above(own),
      });
    }
  }
  const spots = placeLabels(asks, obstacles, { x0: 0, x1: width, y0: 0, y1: AXIS });
  const axisTicks = thinTicks(
    groups.map((group) => ({
      x: group.centre,
      text: tickText(chart.ticks[group.index]),
      cls: group.marked ? ("lb" as const) : ("tk" as const),
      first: group.marked,
    })),
    { x0: 0, x1: width },
  );

  return (
    <>
      <line className="ax" x1={left} y1={AXIS + 0.5} x2={right} y2={AXIS + 0.5} />
      {groups.map((group) =>
        group.bars.map((bar) =>
          bar ? (
            <rect
              key={`${group.index}-${bar.at}`}
              className={cn("bar", bar.was ? "was" : group.marked && "w")}
              x={bar.x}
              y={bar.top}
              width={barWidth}
              height={Math.max(bar.size, 1)}
              rx={radius(bar.size)}
            />
          ) : null,
        ),
      )}
      {asks.map((ask, index) => {
        const spot = spots[index];
        return spot ? (
          // oxlint-disable-next-line react/no-array-index-key -- labels keep their priority order
          <text
            key={index}
            className={ask.className}
            x={spot.x}
            y={spot.y}
            textAnchor={spot.anchor}
          >
            {ask.text}
          </text>
        ) : null;
      })}
      {axisTicks.map(({ index, spot }) => (
        <text
          key={`x${index}`}
          className={groups[index].marked ? "lb w" : "tk"}
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

// ------------------------------------------------------------------ stacked

/** The room beside the last column for the series' names: the longest, within bounds. */
const KEY_CHAR = 6.4;
const KEY_ROOM = { least: 66, most: 140 };

function StackedMarks({ chart, width }: { chart: DeskBoardChart; width: number }) {
  const series = chart.series.map((entry) =>
    entry.points.map((point) => Math.max(0, toNumber(point) ?? 0)),
  );
  const count = Math.max(chart.ticks.length, ...series.map((values) => values.length));
  const totals = Array.from({ length: count }, (_, index) =>
    series.reduce((sum, values) => sum + (values[index] ?? 0), 0),
  );
  const y = linear(0, Math.max(...totals, 0), AXIS, PLOT_TOP);
  const left = 8;
  const keyRoom = Math.min(
    KEY_ROOM.most,
    Math.max(
      KEY_ROOM.least,
      Math.max(0, ...chart.series.map((entry) => entry.name.length)) * KEY_CHAR + 18,
    ),
  );
  const right = width - keyRoom;
  const step = (right - left) / Math.max(1, count);
  const barWidth = Math.max(6, Math.min(24, step - 20));
  const emphasised = chart.marked.length > 0 ? chart.marked[0] : count - 1;
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const lastCentre = left + step * (count - 0.5);

  const segments = (index: number) =>
    series.map((values, at) => {
      const value = values[index] ?? 0;
      const base = series.slice(0, at).reduce((sum, below) => sum + (below[index] ?? 0), 0);
      const bottom = y(base) - (at > 0 ? 2 : 0);
      const top = y(base + value);
      return { at, value, top, size: Math.max(0, bottom - top) };
    });

  const centreOf = (index: number) => left + step * (index + 0.5);
  const obstacles: Box[] = Array.from({ length: count }, (_, index) =>
    segments(index)
      .filter((segment) => segment.value > 0)
      .map((segment) => ({
        x0: centreOf(index) - barWidth / 2,
        x1: centreOf(index) + barWidth / 2,
        y0: segment.top,
        y1: segment.top + segment.size,
      })),
  ).flat();
  const asks: LabelAsk[] = [];
  if (emphasised >= 0 && emphasised < count) {
    asks.push({
      text: format(totals[emphasised]),
      cls: "lb",
      spots: [
        { x: centreOf(emphasised), y: y(totals[emphasised]) - 7, anchor: "middle" },
        { x: centreOf(emphasised) + barWidth / 2, y: y(totals[emphasised]) - 7, anchor: "end" },
        { x: centreOf(emphasised) - barWidth / 2, y: y(totals[emphasised]) - 7, anchor: "start" },
      ],
    });
  }
  const keys = segments(count - 1).filter(
    (segment) => chart.series[segment.at].name !== "" && segment.value > 0,
  );
  for (const segment of keys) {
    asks.push({
      text: fitText(chart.series[segment.at].name, keyRoom - 12, KEY_CHAR),
      cls: "tk",
      spots: [
        {
          x: lastCentre + barWidth / 2 + 6,
          y: segment.top + segment.size / 2 + 4,
          anchor: "start",
        },
      ],
    });
  }
  const spots = placeLabels(asks, obstacles, { x0: 0, x1: width, y0: 0, y1: AXIS });
  const axisTicks = thinTicks(
    Array.from({ length: count }, (_, index) => ({
      x: centreOf(index),
      text: tickText(chart.ticks[index]),
      cls: index === emphasised ? ("lb" as const) : ("tk" as const),
      first: index === emphasised,
    })),
    { x0: 0, x1: right },
  );

  return (
    <>
      <line className="ax" x1={left} y1={AXIS + 0.5} x2={right - 6} y2={AXIS + 0.5} />
      {Array.from({ length: count }, (_, index) =>
        segments(index)
          .filter((segment) => segment.value > 0)
          .map((segment) => (
            <rect
              key={`${index}-${segment.at}`}
              className={cn(
                "bar",
                segment.at === 0 ? index === emphasised && "now" : segment.at === 1 ? "was" : "gh",
              )}
              x={centreOf(index) - barWidth / 2}
              y={segment.top}
              width={barWidth}
              height={segment.size}
              rx={radius(segment.size)}
            />
          )),
      )}
      {asks.map((ask, index) => {
        const spot = spots[index];
        return spot ? (
          // oxlint-disable-next-line react/no-array-index-key -- labels keep their priority order
          <text key={index} className={ask.cls} x={spot.x} y={spot.y} textAnchor={spot.anchor}>
            {ask.text}
          </text>
        ) : null;
      })}
      {axisTicks.map(({ index, spot }) => (
        <text
          key={`x${index}`}
          className={index === emphasised ? "lb" : "tk"}
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

// ------------------------------------------------------------------ labels

/** What a bars chart shows and its headline number, for the chart's label. */
export function barsLabel(chart: DeskBoardChart, title: string, t: TranslateFn): string {
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const layout = barsLayout(chart);
  if (layout === "ranked") {
    const numbered = chart.rows
      .map((row) => ({ row, value: rowNumber(row) }))
      .filter((entry): entry is { row: DeskBoardChartRow; value: number } => entry.value !== null);
    const top =
      numbered.find((entry) => entry.row.marked) ??
      numbered.reduce<(typeof numbered)[number] | null>(
        (best, entry) => (best === null || entry.value > best.value ? entry : best),
        null,
      );
    if (!top) return title;
    return t(
      "{0}: {1} at {2}, the most of {3, plural, one {# row} other {# rows}}",
      title,
      top.row.label,
      top.row.value ? formatBoardValue(top.row.value) : format(top.value),
      chart.rows.length,
    );
  }
  const at =
    chart.marked.length > 0
      ? chart.marked[0]
      : Math.max(chart.ticks.length, ...chart.series.map((series) => series.points.length)) - 1;
  const where = tickText(chart.ticks[at]);
  if (layout === "stacked") {
    const total = chart.series.reduce(
      (sum, series) => sum + Math.max(0, toNumber(series.points[at]) ?? 0),
      0,
    );
    return t("{0}: {1} at {2}", title, format(total), where);
  }
  const now = chart.series.find((series) => series.role === "now") ?? chart.series.at(-1);
  const was = chart.series.find((series) => series !== now);
  const nowValue = now ? toNumber(now.points[at]) : null;
  const wasValue = was ? toNumber(was.points[at]) : null;
  if (nowValue === null) return title;
  if (wasValue === null) return t("{0}: {1} at {2}", title, format(nowValue), where);
  return t("{0}: {1} from {2} to {3}", title, where, format(wasValue), format(nowValue));
}

/**
 * Bars: ranked rows (a name, a bar, its value; the marked one warn), period against period
 * (the earlier period light beside the dark, the marked group warn with both values written),
 * or stacked periods (the first series below, the rest lighter above, the emphasised column
 * dark with its total, the series named beside the last column instead of a legend).
 */
export function BarsChart({ chart, title }: { chart: DeskBoardChart; title: string }) {
  const t = useT();
  const layout = barsLayout(chart);
  return (
    <ChartFrame height={barsHeight(chart)} label={barsLabel(chart, title, t)}>
      {(width) =>
        layout === "ranked" ? (
          <RankedMarks chart={chart} width={width} t={t} />
        ) : layout === "stacked" ? (
          <StackedMarks chart={chart} width={width} />
        ) : (
          <PeriodMarks chart={chart} width={width} t={t} />
        )
      }
    </ChartFrame>
  );
}
