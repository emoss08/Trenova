import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useId } from "react";
import { ChartFrame } from "./chart-frame";
import { CHART_HEIGHTS, chartValue, linear, svgId, tickText, toNumber } from "./chart-values";
import { nowSeries } from "./chart-trend";
import { placeLabels, thinTicks, type Box, type LabelAsk } from "./chart-labels";
import type { DeskBoardChart } from "./metric-types";

const H = CHART_HEIGHTS.columns;
const AXIS = 110;
const TOP = 32;
const TICK_Y = H - 8;

function valuesOf(chart: DeskBoardChart): (number | null)[] {
  return (nowSeries(chart)?.points ?? []).map((point) => toNumber(point));
}

/** Whether columns have a column to draw. */
export function columnsDraw(chart: DeskBoardChart): boolean {
  return valuesOf(chart).some((value) => value !== null);
}

/** What the columns show and their headline number: the longest, and the allowance. */
export function columnsLabel(chart: DeskBoardChart, title: string, t: TranslateFn): string {
  const values = valuesOf(chart);
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const top =
    chart.marked.find((index) => values[index] !== null && values[index] !== undefined) ??
    values.reduce<number>(
      (best, value, index) =>
        value !== null && (best === -1 || value > (values[best] ?? -Infinity)) ? index : best,
      -1,
    );
  if (top === -1) return title;
  const value = format(values[top] ?? 0);
  const where = tickText(chart.ticks[top]);
  if (!chart.ref) return t("{0}: the most {1}, {2}", title, value, where);
  return t(
    "{0}: the most {1}, {2}, against {3} allowed",
    title,
    value,
    where,
    format(Number(chart.ref.value)),
  );
}

function ColumnsMarks({
  chart,
  width,
  hatch,
  t,
}: {
  chart: DeskBoardChart;
  width: number;
  hatch: string;
  t: TranslateFn;
}) {
  const values = valuesOf(chart);
  const count = Math.max(values.length, chart.ticks.length);
  const allowance = chart.ref ? Number(chart.ref.value) : null;
  const known = values.filter((value): value is number => value !== null);
  const low = Math.min(0, ...known);
  const high = Math.max(0, allowance ?? 0, ...known);
  const y = linear(low, high, AXIS, TOP);
  const zero = y(0);
  const left = 20;
  const right = width - 8;
  const step = (right - left) / Math.max(1, count);
  const barWidth = Math.max(6, Math.min(30, step * 0.55));
  const marked = new Set(chart.marked);
  const format = (value: number) => chartValue(value, chart.unit, chart.currency);
  const line = allowance === null ? null : y(allowance);

  const columns = Array.from({ length: count }, (_, index) => {
    const value = values[index];
    const centre = left + step * (index + 0.5);
    if (value === null || value === undefined) {
      return { index, centre, value: null, solidTop: 0, solidSize: 0, hatchTop: 0, hatchSize: 0 };
    }
    const over = line !== null && allowance !== null && value > allowance;
    const solidTop = over && line !== null ? line + 1 : Math.min(y(value), zero);
    const solidSize = over ? zero - solidTop : Math.abs(y(value) - zero);
    const hatchTop = y(value);
    const hatchSize = over && line !== null ? line - 1 - hatchTop : 0;
    return { index, centre, value, solidTop, solidSize, hatchTop, hatchSize };
  });
  const obstacles: Box[] = columns.flatMap((column) =>
    column.value === null
      ? []
      : [
          {
            x0: column.centre - barWidth / 2,
            x1: column.centre + barWidth / 2,
            y0: Math.min(column.solidTop, column.hatchSize > 0 ? column.hatchTop : column.solidTop),
            y1: column.solidTop + Math.max(column.solidSize, 1),
          },
        ],
  );
  const refText =
    chart.ref === null
      ? ""
      : chart.ref.label === ""
        ? format(Number(chart.ref.value))
        : t("{0} {1}", chart.ref.label, format(Number(chart.ref.value)));
  const asks: LabelAsk[] = [];
  if (line !== null && refText !== "") {
    asks.push({
      text: refText,
      cls: "tk",
      spots: [
        { x: 0, y: line - 5, anchor: "start" },
        { x: right, y: line - 5, anchor: "end" },
        { x: 0, y: line + 13, anchor: "start" },
        { x: right, y: line + 13, anchor: "end" },
      ],
    });
  }
  const valued = columns.filter((column) => marked.has(column.index) && column.value !== null);
  for (const column of valued) {
    const top = column.hatchSize > 0 ? column.hatchTop : column.solidTop;
    asks.push({
      text: format(column.value ?? 0),
      cls: "lb",
      spots: [
        { x: column.centre, y: top - 7, anchor: "middle" },
        { x: column.centre - barWidth / 2, y: top - 7, anchor: "start" },
        { x: column.centre + barWidth / 2, y: top - 7, anchor: "end" },
      ],
    });
  }
  const spots = placeLabels(asks, obstacles, { x0: 0, x1: width, y0: 0, y1: AXIS });
  const refSpot = line !== null && refText !== "" ? spots[0] : null;
  const valueSpots = spots.slice(line !== null && refText !== "" ? 1 : 0);
  const axisTicks = thinTicks(
    columns.map((column) => ({
      x: column.centre,
      text: tickText(chart.ticks[column.index]),
      cls: marked.has(column.index) ? ("lb" as const) : ("tk" as const),
      first: marked.has(column.index),
    })),
    { x0: 0, x1: width },
  );

  return (
    <>
      {line !== null && (
        <defs>
          <pattern
            id={hatch}
            width={6}
            height={6}
            patternUnits="userSpaceOnUse"
            patternTransform="rotate(45)"
          >
            <rect className="hp-b" width={6} height={6} />
            <rect className="hp-a" width={3} height={6} />
          </pattern>
        </defs>
      )}
      <line className="ax" x1={left} y1={AXIS + 0.5} x2={right} y2={AXIS + 0.5} />
      {columns.map((column) =>
        column.value === null ? null : (
          <g key={column.index}>
            <rect
              className="bar"
              x={column.centre - barWidth / 2}
              y={column.solidTop}
              width={barWidth}
              height={Math.max(column.solidSize, 1)}
              rx={Math.min(2, column.solidSize / 2)}
            />
            {column.hatchSize > 0 && (
              <rect
                x={column.centre - barWidth / 2}
                y={column.hatchTop}
                width={barWidth}
                height={column.hatchSize}
                rx={Math.min(2, column.hatchSize / 2)}
                fill={`url(#${hatch})`}
              />
            )}
          </g>
        ),
      )}
      {line !== null && <line className="ref" x1={left} y1={line} x2={right} y2={line} />}
      {refSpot && (
        <text className="tk" x={refSpot.x} y={refSpot.y} textAnchor={refSpot.anchor}>
          {refText}
        </text>
      )}
      {valued.map((column, index) => {
        const spot = valueSpots[index];
        return spot ? (
          <text
            key={`v${column.index}`}
            className="lb w"
            x={spot.x}
            y={spot.y}
            textAnchor={spot.anchor}
          >
            {format(column.value ?? 0)}
          </text>
        ) : null;
      })}
      {axisTicks.map(({ index, spot }) => (
        <text
          key={`x${index}`}
          className={marked.has(index) ? "lb w" : "tk"}
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
 * Columns over an allowance: each column solid up to the allowance and the warn hatch past it,
 * the allowance a dashed reference with its value, the marked column's value written in warn.
 */
export function ColumnsChart({ chart, title }: { chart: DeskBoardChart; title: string }) {
  const t = useT();
  const hatch = svgId(useId());
  return (
    <ChartFrame height={H} label={columnsLabel(chart, title, t)}>
      {(width) => <ColumnsMarks chart={chart} width={width} hatch={hatch} t={t} />}
    </ChartFrame>
  );
}
