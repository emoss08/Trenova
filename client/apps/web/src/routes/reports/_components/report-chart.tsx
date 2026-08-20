import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { cn } from "@trenova/shared/lib/utils";
import { ReportMap } from "./report-map";
import {
  formatReportValue,
  reportNumericValue,
  type ReportColumnDisplayLike,
} from "@trenova/shared/lib/report-format";
import type { ReportChartSpec } from "@/types/report";
import { TrendingDownIcon, TrendingUpIcon } from "lucide-react";
import { useMemo } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  LabelList,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ReferenceLine,
  Scatter,
  ScatterChart,
  XAxis,
  YAxis,
  ZAxis,
} from "recharts";

const PALETTE = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
];

const LABEL_KEY = "__label";
/** The unformatted category value, so a click can filter on what the cell held. */
const RAW_KEY = "__raw";
const MAX_PLOTTED_CATEGORIES = 200;

export type ChartColumn = {
  id: string;
  label: string;
  display?: ReportColumnDisplayLike | null;
};

type ReportChartProps = {
  chart: ReportChartSpec;
  columns: ChartColumn[];
  rows: unknown[][];
  className?: string;
  /** Enables click-to-filter on the category axis. */
  categorySelect?: CategorySelect;
};

type ChartPoint = Record<string, unknown>;

/** Notified with the raw category value when a chart category is clicked. */
export type CategorySelect = {
  onSelect: (raw: unknown, label: string) => void;
  /** The currently filtered category, so the chart can show what is active. */
  selected?: unknown;
};

function seriesColor(index: number): string {
  return PALETTE[index % PALETTE.length];
}

function columnIndexMap(columns: ChartColumn[]): Map<string, number> {
  return new Map(columns.map((column, index) => [column.id, index]));
}

/**
 * Flattens the result rows into what recharts wants: one object per category,
 * with each plotted measure as a numeric key. Non-numeric cells become null so
 * a gap reads as missing data rather than as zero.
 */
function buildPoints(
  chart: ReportChartSpec,
  columns: ChartColumn[],
  rows: unknown[][],
  indexes: Map<string, number>,
): ChartPoint[] {
  const labelIndex = chart.xColumnId ? indexes.get(chart.xColumnId) : undefined;
  const labelColumn = labelIndex === undefined ? undefined : columns[labelIndex];
  const seriesIds = chart.seriesIds ?? [];

  const points = rows.map((row, rowIndex) => {
    const point: ChartPoint = {
      [LABEL_KEY]:
        labelIndex === undefined
          ? `#${rowIndex + 1}`
          : formatReportValue(row[labelIndex], labelColumn?.display) || "—",
      [RAW_KEY]: labelIndex === undefined ? null : row[labelIndex],
    };
    for (const seriesId of seriesIds) {
      const columnIndex = indexes.get(seriesId);
      point[seriesId] = columnIndex === undefined ? null : reportNumericValue(row[columnIndex]);
    }
    if (chart.type === "scatter" && chart.xColumnId) {
      const xIndex = indexes.get(chart.xColumnId);
      point.x = xIndex === undefined ? null : reportNumericValue(row[xIndex]);
    }
    return point;
  });

  const primary = seriesIds[0];
  if (chart.limit && chart.limit > 0 && primary) {
    return [...points]
      .sort((a, b) => Number(b[primary] ?? 0) - Number(a[primary] ?? 0))
      .slice(0, chart.limit);
  }

  return points.slice(0, MAX_PLOTTED_CATEGORIES);
}

/**
 * Axis ticks and bar labels have to fit in a few characters, so they keep the
 * column's unit (currency symbol, suffix) but drop to compact notation.
 */
function compactDisplay(
  display: ReportColumnDisplayLike | null | undefined,
): ReportColumnDisplayLike | null | undefined {
  if (!display) return display;
  return {
    ...display,
    notation: "compact",
    decimals: Math.min(display.decimals ?? 1, 1),
  };
}

export type ChartFormatters = {
  /** Full precision, for tooltips. */
  value: (seriesId: string, value: unknown) => string;
  label: (seriesId: string) => string;
  /** Compact, for axis ticks and on-bar labels. */
  axis: (value: unknown) => string;
  xAxis: (value: unknown) => string;
};

// Charts render the same numbers the table does, so they read the column's
// resolved display rather than dumping the raw value onto the axis.
function buildFormatters(chart: ReportChartSpec, columns: ChartColumn[]): ChartFormatters {
  const byId = new Map(columns.map((column) => [column.id, column]));
  const primary = chart.seriesIds?.[0];
  const primaryDisplay = primary ? byId.get(primary)?.display : undefined;
  const xDisplay = chart.xColumnId ? byId.get(chart.xColumnId)?.display : undefined;

  return {
    value: (seriesId, value) => formatReportValue(value, byId.get(seriesId)?.display),
    label: (seriesId) => byId.get(seriesId)?.label ?? seriesId,
    axis: (value) => formatReportValue(value, compactDisplay(primaryDisplay)),
    xAxis: (value) => formatReportValue(value, compactDisplay(xDisplay)),
  };
}

function chartConfig(chart: ReportChartSpec, columns: ChartColumn[]): ChartConfig {
  const byId = new Map(columns.map((column) => [column.id, column]));
  const config: ChartConfig = {};
  (chart.seriesIds ?? []).forEach((seriesId, index) => {
    config[seriesId] = {
      label: byId.get(seriesId)?.label ?? seriesId,
      color: seriesColor(index),
    };
  });
  return config;
}

function goalValue(
  chart: ReportChartSpec,
  rows: unknown[][],
  indexes: Map<string, number>,
): number | null {
  if (!chart.goal) return null;
  if (chart.goal.value !== undefined && chart.goal.value !== null) return chart.goal.value;
  if (!chart.goal.columnId) return null;

  const columnIndex = indexes.get(chart.goal.columnId);
  if (columnIndex === undefined) return null;

  // A goal column repeats the same target on every row, so the first row that
  // carries a number is the target for the whole chart.
  for (const row of rows) {
    const value = reportNumericValue(row[columnIndex]);
    if (value !== null) return value;
  }
  return null;
}

/**
 * The default tooltip row lets the label and the value touch once the box
 * sizes to its content, which a long currency figure always does. Rendering
 * the row here keeps a fixed gap and applies the column's formatting.
 */
function tooltipRow(format: ChartFormatters, nameToSeries?: (name: string) => string) {
  return (value: unknown, name: unknown, item: { color?: string }) => {
    // Recharts types `name` loosely, but a series name is always the dataKey
    // string (or, for a pie, the category label).
    const key = typeof name === "string" || typeof name === "number" ? String(name) : "";
    const seriesId = nameToSeries ? nameToSeries(key) : key;
    return (
      <>
        <span className="size-2.5 shrink-0 rounded-[2px]" style={{ background: item?.color }} />
        <span className="text-muted-foreground flex-1 truncate">
          {nameToSeries ? key : format.label(seriesId)}
        </span>
        <span className="text-foreground ml-4 font-mono font-medium tabular-nums">
          {format.value(seriesId, value)}
        </span>
      </>
    );
  };
}

/** Recharts hands a click either the datum or a wrapper carrying it. */
function pointFromClick(data: unknown): ChartPoint | null {
  if (!data || typeof data !== "object") return null;
  const wrapper = data as { payload?: unknown };
  const point = (wrapper.payload ?? data) as ChartPoint;
  return typeof point === "object" && point !== null ? point : null;
}

/** Category cells are primitives, so identity compares on their text form. */
function categoryText(value: unknown): string | null {
  if (value === null || value === undefined) return null;
  const primitive =
    typeof value === "string" || typeof value === "number" || typeof value === "boolean";
  return primitive ? String(value) : null;
}

function sameCategory(a: unknown, b: unknown): boolean {
  const left = categoryText(a);
  const right = categoryText(b);
  return left !== null && left === right;
}

function categoryClickProps(select: CategorySelect | undefined) {
  if (!select) return {};
  return {
    cursor: "pointer",
    onClick: (data: unknown) => {
      const point = pointFromClick(data);
      if (!point) return;
      select.onSelect(point[RAW_KEY], categoryText(point[LABEL_KEY]) ?? "");
    },
  };
}

/** Dims the categories a click did not select, so the filter is visible. */
function selectionCells(points: ChartPoint[], select: CategorySelect | undefined, fill?: string) {
  if (!select || select.selected === undefined) return null;
  return points.map((point, pointIndex) => (
    <Cell
      key={pointIndex}
      fill={fill ?? seriesColor(pointIndex)}
      fillOpacity={sameCategory(point[RAW_KEY], select.selected) ? 1 : 0.25}
    />
  ));
}

function EmptyChart({ message }: { message: string }) {
  return (
    <div className="text-muted-foreground flex h-full min-h-32 w-full items-center justify-center p-4 text-center text-xs">
      {message}
    </div>
  );
}

function KpiTile({
  chart,
  columns,
  rows,
  indexes,
  className,
}: ReportChartProps & { indexes: Map<string, number> }) {
  const seriesId = chart.seriesIds?.[0];
  const byId = new Map(columns.map((column) => [column.id, column]));
  const column = seriesId ? byId.get(seriesId) : undefined;
  const columnIndex = seriesId ? indexes.get(seriesId) : undefined;

  if (!column || columnIndex === undefined) {
    return <EmptyChart message="Choose the measure this KPI shows." />;
  }

  const row = rows[0] ?? [];
  const value = row[columnIndex];
  const current = reportNumericValue(value);

  const compareIndex = chart.compareId ? indexes.get(chart.compareId) : undefined;
  const compareColumn = chart.compareId ? byId.get(chart.compareId) : undefined;
  const previous = compareIndex === undefined ? null : reportNumericValue(row[compareIndex]);
  const delta =
    current !== null && previous !== null && previous !== 0
      ? (current - previous) / Math.abs(previous)
      : null;

  const target = goalValue(chart, rows, indexes);
  const attainment = current !== null && target !== null && target !== 0 ? current / target : null;

  return (
    <div className={cn("flex h-full flex-col justify-center gap-1 p-4", className)}>
      <p className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
        {chart.title || column.label}
      </p>
      <p className="text-2xl font-semibold tracking-tight tabular-nums">
        {formatReportValue(value, column.display) || "—"}
      </p>
      {delta !== null && (
        <p
          className={cn(
            "flex items-center gap-1 text-xs tabular-nums",
            delta >= 0 ? "text-emerald-600 dark:text-emerald-400" : "text-destructive",
          )}
        >
          {delta >= 0 ? (
            <TrendingUpIcon className="size-3.5" />
          ) : (
            <TrendingDownIcon className="size-3.5" />
          )}
          {(delta * 100).toFixed(1)}%
          <span className="text-muted-foreground">vs {compareColumn?.label ?? "previous"}</span>
        </p>
      )}
      {attainment !== null && (
        <div className="flex flex-col gap-1">
          <div className="bg-muted h-1.5 overflow-hidden rounded-full">
            <div
              className="bg-primary h-full rounded-full transition-[width] duration-500"
              style={{ width: `${Math.min(Math.max(attainment, 0), 1) * 100}%` }}
            />
          </div>
          <p className="text-2xs text-muted-foreground tabular-nums">
            {(attainment * 100).toFixed(0)}% of {chart.goal?.label || "target"}
          </p>
        </div>
      )}
    </div>
  );
}

export function ReportChart(props: ReportChartProps) {
  const { chart, columns, rows, className, categorySelect } = props;
  const indexes = useMemo(() => columnIndexMap(columns), [columns]);
  const points = useMemo(
    () => buildPoints(chart, columns, rows, indexes),
    [chart, columns, rows, indexes],
  );
  const config = useMemo(() => chartConfig(chart, columns), [chart, columns]);
  const format = useMemo(() => buildFormatters(chart, columns), [chart, columns]);
  const target = goalValue(chart, rows, indexes);
  const seriesIds = chart.seriesIds ?? [];

  if (chart.type === "kpi") {
    return <KpiTile {...props} indexes={indexes} />;
  }
  // A map plots raw rows at coordinates rather than a Recharts series, so it
  // bypasses the shared point/series pipeline entirely.
  if (chart.type === "map") {
    return <ReportMap chart={chart} columns={columns} rows={rows} className={className} />;
  }
  if (seriesIds.length === 0) {
    return <EmptyChart message="Choose at least one measure to plot." />;
  }
  if (points.length === 0) {
    return <EmptyChart message="This report returned no rows to plot." />;
  }

  const legend = !chart.hideLegend && seriesIds.length > 1;
  const stackId = chart.stacked ? "stack" : undefined;

  // Recharts measures its parent, and a chart box that only inherits `h-full`
  // through a flex chain collapses to zero. Filling an absolutely positioned
  // box gives it a definite size wherever it is dropped.
  return (
    <div className={cn("relative h-full w-full", className)}>
      <div className="absolute inset-0">
        <ChartContainer config={config} className="aspect-auto h-full w-full">
          {renderChart({
            chart,
            points,
            seriesIds,
            legend,
            stackId,
            target,
            columns,
            indexes,
            format,
            categorySelect,
          })}
        </ChartContainer>
      </div>
    </div>
  );
}

type RenderArgs = {
  chart: ReportChartSpec;
  points: ChartPoint[];
  seriesIds: string[];
  legend: boolean;
  stackId: string | undefined;
  target: number | null;
  columns: ChartColumn[];
  indexes: Map<string, number>;
  format: ChartFormatters;
  categorySelect?: CategorySelect;
};

function renderChart(args: RenderArgs) {
  switch (args.chart.type) {
    case "line":
      return renderLine(args);
    case "area":
      return renderArea(args);
    case "pie":
    case "donut":
      return renderPie(args);
    case "scatter":
      return renderScatter(args);
    case "hbar":
    case "bar":
    default:
      return renderBar(args);
  }
}

const AXIS_PROPS = {
  tickLine: false,
  axisLine: false,
  tickMargin: 8,
} as const;

function GoalLine({ target, label }: { target: number | null; label?: string }) {
  if (target === null) return null;
  return (
    <ReferenceLine
      y={target}
      stroke="var(--primary)"
      strokeDasharray="4 4"
      label={{ value: label || "Target", position: "right", fontSize: 10 }}
    />
  );
}

function renderBar({
  chart,
  points,
  seriesIds,
  legend,
  stackId,
  target,
  format,
  categorySelect,
}: RenderArgs) {
  const horizontal = chart.type === "hbar";
  // Only a single-series chart can attribute a click to one category.
  const select = seriesIds.length === 1 ? categorySelect : undefined;

  return (
    <BarChart data={points} layout={horizontal ? "vertical" : "horizontal"}>
      <CartesianGrid vertical={horizontal} horizontal={!horizontal} strokeDasharray="3 3" />
      {horizontal ? (
        <>
          <XAxis type="number" tickFormatter={format.axis} {...AXIS_PROPS} />
          <YAxis type="category" dataKey={LABEL_KEY} width={120} {...AXIS_PROPS} />
        </>
      ) : (
        <>
          <XAxis dataKey={LABEL_KEY} {...AXIS_PROPS} interval="preserveStartEnd" />
          <YAxis tickFormatter={format.axis} {...AXIS_PROPS} width={72} />
        </>
      )}
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      {!horizontal && <GoalLine target={target} label={chart.goal?.label} />}
      {seriesIds.map((seriesId, index) => (
        <Bar
          key={seriesId}
          dataKey={seriesId}
          stackId={stackId}
          fill={seriesColor(index)}
          radius={horizontal ? [0, 3, 3, 0] : [3, 3, 0, 0]}
          isAnimationActive
          {...categoryClickProps(select)}
        >
          {selectionCells(points, select, seriesColor(index))}
          {chart.showValues && (
            <LabelList
              dataKey={seriesId}
              position={horizontal ? "right" : "top"}
              formatter={format.axis}
              className="fill-muted-foreground"
              fontSize={10}
            />
          )}
        </Bar>
      ))}
    </BarChart>
  );
}

function renderLine({ chart, points, seriesIds, legend, target, format }: RenderArgs) {
  return (
    <LineChart data={points}>
      <CartesianGrid vertical={false} strokeDasharray="3 3" />
      <XAxis dataKey={LABEL_KEY} {...AXIS_PROPS} interval="preserveStartEnd" />
      <YAxis tickFormatter={format.axis} {...AXIS_PROPS} width={72} />
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      <GoalLine target={target} label={chart.goal?.label} />
      {seriesIds.map((seriesId, index) => (
        <Line
          key={seriesId}
          dataKey={seriesId}
          type={chart.curved ? "monotone" : "linear"}
          stroke={seriesColor(index)}
          strokeWidth={2}
          dot={false}
          connectNulls
        />
      ))}
    </LineChart>
  );
}

function renderArea({ chart, points, seriesIds, legend, stackId, target, format }: RenderArgs) {
  return (
    <AreaChart data={points}>
      <CartesianGrid vertical={false} strokeDasharray="3 3" />
      <XAxis dataKey={LABEL_KEY} {...AXIS_PROPS} interval="preserveStartEnd" />
      <YAxis tickFormatter={format.axis} {...AXIS_PROPS} width={72} />
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      <GoalLine target={target} label={chart.goal?.label} />
      {seriesIds.map((seriesId, index) => (
        <Area
          key={seriesId}
          dataKey={seriesId}
          type={chart.curved ? "monotone" : "linear"}
          stackId={stackId}
          stroke={seriesColor(index)}
          fill={seriesColor(index)}
          fillOpacity={0.18}
          strokeWidth={2}
          connectNulls
        />
      ))}
    </AreaChart>
  );
}

function renderPie({ chart, points, seriesIds, legend, format, categorySelect }: RenderArgs) {
  const seriesId = seriesIds[0];

  return (
    <PieChart>
      {/* A slice is named by its category, so the value is formatted with the
          one measure the pie plots rather than by looking the name up. */}
      <ChartTooltip
        content={
          <ChartTooltipContent nameKey={LABEL_KEY} formatter={tooltipRow(format, () => seriesId)} />
        }
      />
      <Pie
        data={points}
        dataKey={seriesId}
        nameKey={LABEL_KEY}
        innerRadius={chart.type === "donut" ? "55%" : 0}
        outerRadius="80%"
        paddingAngle={1}
        {...categoryClickProps(categorySelect)}
      >
        {points.map((point, index) => (
          <Cell
            key={index}
            fill={seriesColor(index)}
            fillOpacity={
              categorySelect?.selected === undefined ||
              sameCategory(point[RAW_KEY], categorySelect.selected)
                ? 1
                : 0.25
            }
          />
        ))}
      </Pie>
      {legend && <Legend />}
    </PieChart>
  );
}

function renderScatter({ chart, points, seriesIds, columns, indexes, format }: RenderArgs) {
  const seriesId = seriesIds[0];
  const xColumn = chart.xColumnId ? columns[indexes.get(chart.xColumnId) ?? -1] : undefined;
  const yColumn = columns[indexes.get(seriesId) ?? -1];

  return (
    <ScatterChart>
      <CartesianGrid strokeDasharray="3 3" />
      <XAxis
        type="number"
        dataKey="x"
        name={xColumn?.label ?? "X"}
        tickFormatter={format.xAxis}
        {...AXIS_PROPS}
      />
      <YAxis
        type="number"
        dataKey={seriesId}
        name={yColumn?.label ?? "Y"}
        tickFormatter={format.axis}
        {...AXIS_PROPS}
      />
      <ZAxis range={[60, 60]} />
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(format)} />} />
      <Scatter data={points} fill={seriesColor(0)} />
    </ScatterChart>
  );
}
