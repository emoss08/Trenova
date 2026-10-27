import {
  CHART_LABEL_KEY,
  CHART_X_KEY,
  analysisChartPoints,
  analysisPieSlices,
  chartHasValues,
  chartSeriesKey,
  type AnalysisChartPoint,
} from "@/lib/analysis-chart";
import { formatAnalysisAxisValue, formatAnalysisValue } from "@/lib/analysis-values";
import type { AnalysisChart } from "@/types/analysis";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
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

const AXIS_PROPS = { tickLine: false, axisLine: false, tickMargin: 8 } as const;

/** A horizontal bar gives each category this much height, so labels never overlap. */
const BAR_ROW_PX = 24;
const MIN_HEIGHT_PX = 256;

function seriesColor(index: number): string {
  return PALETTE[index % PALETTE.length];
}

type Formatters = {
  value: (value: unknown) => string;
  axis: (value: unknown) => string;
  x: (value: unknown) => string;
};

function formattersOf(chart: AnalysisChart, t: TranslateFn): Formatters {
  return {
    value: (value) => formatAnalysisValue(value, chart.valueKind, t, chart.currency),
    axis: (value) => formatAnalysisAxisValue(value, chart.valueKind, chart.currency),
    x: (value) => formatAnalysisAxisValue(value, chart.x.kind),
  };
}

function configOf(chart: AnalysisChart): ChartConfig {
  const config: ChartConfig = {};
  chart.series.forEach((series, index) => {
    config[chartSeriesKey(index)] = { label: series.name, color: seriesColor(index) };
  });
  return config;
}

/** A tooltip row with a fixed gap between the series and its value, read in the chart's kind. */
function tooltipRow(chart: AnalysisChart, format: Formatters, byCategory = false) {
  return (value: unknown, name: unknown, item: { color?: string; payload?: unknown }) => {
    const key = typeof name === "string" ? name : "";
    const index = chart.series.findIndex((_, seriesIndex) => chartSeriesKey(seriesIndex) === key);
    const point = (item.payload ?? {}) as AnalysisChartPoint;
    const label = byCategory
      ? String(point[CHART_LABEL_KEY] ?? "")
      : (chart.series[index]?.name ?? key);
    return (
      <>
        <span className="size-2.5 shrink-0 rounded-[2px]" style={{ background: item.color }} />
        <span className="text-muted-foreground flex-1 truncate">{label}</span>
        <span className="text-foreground ml-4 font-mono font-medium tabular-nums">
          {format.value(value)}
        </span>
      </>
    );
  };
}

/**
 * A chart a script showed, drawn from what it plots in the reader's theme: bars, lines, areas,
 * points or slices, with its values read as money, counts, percents or numbers. A chart with no
 * number to draw says so instead of drawing an empty frame.
 */
export function AnalysisChartView({ chart }: { chart: AnalysisChart }) {
  const t = useT();
  const title = chart.title || t("Chart");

  if (!chartHasValues(chart)) {
    return (
      <figure aria-label={title} className="flex flex-col gap-1.5">
        {chart.title !== "" && (
          <figcaption className="text-sm font-medium">{chart.title}</figcaption>
        )}
        <p className="text-muted-foreground rounded-surface border p-3 text-xs">
          {t("This chart has no values to draw.")}
        </p>
      </figure>
    );
  }

  const points = analysisChartPoints(chart, t);
  const format = formattersOf(chart, t);
  const height =
    chart.kind === "bar" && chart.horizontal
      ? Math.max(MIN_HEIGHT_PX, points.length * BAR_ROW_PX)
      : MIN_HEIGHT_PX;
  const legend = chart.series.length > 1;

  return (
    <figure aria-label={title} className="flex flex-col gap-1.5">
      {chart.title !== "" && <figcaption className="text-sm font-medium">{chart.title}</figcaption>}
      <div className="rounded-surface border p-2">
        <ChartContainer config={configOf(chart)} className="aspect-auto w-full" style={{ height }}>
          {renderChart({ chart, points, format, legend, t })}
        </ChartContainer>
      </div>
      {chart.kind === "pie" && <PieLeftOut points={points} />}
    </figure>
  );
}

function PieLeftOut({ points }: { points: AnalysisChartPoint[] }) {
  const t = useT();
  const { left } = analysisPieSlices(points);
  if (left === 0) {
    return null;
  }

  return (
    <p className="text-muted-foreground text-xs">
      {t(
        "{0, plural, one {# value is} other {# values are}} not drawn: a pie shows only values above zero.",
        left,
      )}
    </p>
  );
}

type RenderArgs = {
  chart: AnalysisChart;
  points: AnalysisChartPoint[];
  format: Formatters;
  legend: boolean;
  t: TranslateFn;
};

function renderChart(args: RenderArgs) {
  switch (args.chart.kind) {
    case "line":
      return renderLine(args);
    case "area":
      return renderArea(args);
    case "scatter":
      return renderScatter(args);
    case "pie":
      return renderPie(args);
    case "bar":
      return renderBar(args);
  }
}

function renderBar({ chart, points, format, legend }: RenderArgs) {
  const stackId = chart.stacked ? "stack" : undefined;
  const horizontal = chart.horizontal;

  return (
    <BarChart data={points} layout={horizontal ? "vertical" : "horizontal"}>
      <CartesianGrid vertical={horizontal} horizontal={!horizontal} strokeDasharray="3 3" />
      {horizontal ? (
        <>
          <XAxis type="number" tickFormatter={format.axis} {...AXIS_PROPS} />
          <YAxis type="category" dataKey={CHART_LABEL_KEY} width={120} {...AXIS_PROPS} />
        </>
      ) : (
        <>
          <XAxis dataKey={CHART_LABEL_KEY} interval="preserveStartEnd" {...AXIS_PROPS} />
          <YAxis tickFormatter={format.axis} width={72} {...AXIS_PROPS} />
        </>
      )}
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(chart, format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      {chart.series.map((_, index) => (
        <Bar
          key={chartSeriesKey(index)}
          dataKey={chartSeriesKey(index)}
          stackId={stackId}
          fill={seriesColor(index)}
          radius={horizontal ? [0, 3, 3, 0] : [3, 3, 0, 0]}
        />
      ))}
    </BarChart>
  );
}

function renderLine({ chart, points, format, legend }: RenderArgs) {
  return (
    <LineChart data={points}>
      <CartesianGrid vertical={false} strokeDasharray="3 3" />
      <XAxis dataKey={CHART_LABEL_KEY} interval="preserveStartEnd" {...AXIS_PROPS} />
      <YAxis tickFormatter={format.axis} width={72} {...AXIS_PROPS} />
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(chart, format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      {chart.series.map((_, index) => (
        <Line
          key={chartSeriesKey(index)}
          dataKey={chartSeriesKey(index)}
          type="linear"
          stroke={seriesColor(index)}
          strokeWidth={2}
          dot={points.length <= 31}
          connectNulls
        />
      ))}
    </LineChart>
  );
}

function renderArea({ chart, points, format, legend }: RenderArgs) {
  const stackId = chart.stacked ? "stack" : undefined;

  return (
    <AreaChart data={points}>
      <CartesianGrid vertical={false} strokeDasharray="3 3" />
      <XAxis dataKey={CHART_LABEL_KEY} interval="preserveStartEnd" {...AXIS_PROPS} />
      <YAxis tickFormatter={format.axis} width={72} {...AXIS_PROPS} />
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(chart, format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      {chart.series.map((_, index) => (
        <Area
          key={chartSeriesKey(index)}
          dataKey={chartSeriesKey(index)}
          type="linear"
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

function renderScatter({ chart, points, format, legend, t }: RenderArgs) {
  return (
    <ScatterChart>
      <CartesianGrid strokeDasharray="3 3" />
      <XAxis
        type="number"
        dataKey={CHART_X_KEY}
        name={chart.x.label || t("X")}
        tickFormatter={format.x}
        {...AXIS_PROPS}
      />
      <YAxis type="number" dataKey="y" tickFormatter={format.axis} width={72} {...AXIS_PROPS} />
      <ZAxis range={[48, 48]} />
      <ChartTooltip content={<ChartTooltipContent formatter={tooltipRow(chart, format)} />} />
      {legend && <ChartLegend content={<ChartLegendContent />} />}
      {chart.series.map((_, index) => {
        const key = chartSeriesKey(index);
        const data = points
          .filter((point) => point[CHART_X_KEY] !== null && point[key] !== null)
          .map((point) => ({ ...point, y: point[key] }));
        return <Scatter key={key} name={key} data={data} fill={seriesColor(index)} />;
      })}
    </ScatterChart>
  );
}

function renderPie({ chart, points, format }: RenderArgs) {
  const { slices } = analysisPieSlices(points);

  return (
    <PieChart>
      <ChartTooltip
        content={
          <ChartTooltipContent
            nameKey={CHART_LABEL_KEY}
            formatter={tooltipRow(chart, format, true)}
          />
        }
      />
      <Pie
        data={slices}
        dataKey={chartSeriesKey(0)}
        nameKey={CHART_LABEL_KEY}
        outerRadius="80%"
        paddingAngle={1}
      >
        {slices.map((slice, index) => (
          <Cell key={`${index}-${String(slice[CHART_LABEL_KEY])}`} fill={seriesColor(index)} />
        ))}
      </Pie>
      <Legend />
    </PieChart>
  );
}
