import type { AgentAnalytics, AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, formatUnixTime } from "@trenova/shared/lib/date";
import { memo, useMemo, useState, type CSSProperties, type MouseEvent } from "react";
import { Seg } from "../kit/layout";
import {
  linePath,
  METRICS,
  stackLabels,
  niceTicks,
  TREND_METRICS,
  bucketPosition,
  trendLines,
  type SeriesColors,
  type TrendMetric,
} from "./analytics-model";
import { AnalyticsCard, Swatch, Tip } from "./card";
import { hoverProps, useSetHot } from "./highlight";

const BASE_HEIGHT = 260;
const EVENT_ROW = 14;
const EVENT_GAP = 12;
const EVENT_CHAR_WIDTH = 6;
const EVENT_LABEL_ROOM = 140;
const TIP_FLIP_ROOM = 240;
const X_LABELS = 6;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86_400;

/** Where an event's label sits: right of its marker, or left of it near the chart's end. */
function labelSpan(x: number, label: string, right: number): { start: number; end: number } {
  const width = label.length * EVENT_CHAR_WIDTH + 7;
  const atEnd = x > right - EVENT_LABEL_ROOM;

  return atEnd ? { start: x - width, end: x } : { start: x, end: x + width };
}

function bucketLabel(
  t: TranslateFn,
  analytics: AgentAnalytics,
  start: number,
  last: number,
): string {
  const ago = last - start;
  if (analytics.window === "Day") {
    return ago <= 0 ? t("Now") : t("{0}h ago", Math.round(ago / SECONDS_PER_HOUR));
  }
  if (analytics.window === "Week") {
    return `${formatUnixMonthDay(start)} ${formatUnixTime(start)}`;
  }

  return ago < SECONDS_PER_DAY ? t("Today") : formatUnixMonthDay(start);
}

type OverTimeCardProps = {
  analytics: AgentAnalytics;
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
};

/** Each series over the window, with the provider changes that might explain a turn marked. */
export const OverTimeCard = memo(function OverTimeCard({
  analytics,
  series,
  colors,
}: OverTimeCardProps) {
  const t = useT();
  const [metricKey, setMetricKey] = useState<TrendMetric>("completed");
  const [hover, setHover] = useState<number | null>(null);
  const [ref, width] = useElementWidth<HTMLDivElement>();
  const setHot = useSetHot();
  const choice = TREND_METRICS.find((item) => item.value === metricKey) ?? TREND_METRICS[0];
  const metric = METRICS[choice?.metric ?? "completed"];

  const buckets = analytics.trendBuckets;
  const lines = useMemo(
    () =>
      trendLines(
        analytics.trend,
        metricKey,
        series.map((item) => item.key),
        buckets,
      ),
    [analytics.trend, buckets, metricKey, series],
  );
  const shown = useMemo(() => {
    const providerIds = new Set(
      series.flatMap((item) => item.providers.map((provider) => provider.id)),
    );
    return analytics.events.filter(
      (event) => event.at >= (buckets[0] ?? 0) && providerIds.has(event.provider.id),
    );
  }, [analytics.events, buckets, series]);

  const count = buckets.length;
  const last = buckets[count - 1] ?? 0;
  const indexOf = (at: number) => bucketPosition(buckets, at, analytics.bucketSeconds);
  const roughWidth = Math.max(10, width - 64);
  const eventX = (at: number) => 48 + (count > 1 ? indexOf(at) / (count - 1) : 0) * roughWidth;
  const rows = stackLabels(
    shown.map((event) => labelSpan(eventX(event.at), event.label, 48 + roughWidth)),
    EVENT_GAP,
  );
  const events = shown.map((event, index) => ({ event, row: rows[index] ?? 0 }));
  const eventRows = events.length ? Math.max(...events.map((item) => item.row)) + 1 : 0;

  const height = BASE_HEIGHT + eventRows * EVENT_ROW;
  const pad = { left: 48, right: 16, top: 18 + eventRows * EVENT_ROW, bottom: 28 };
  const innerWidth = Math.max(10, width - pad.left - pad.right);
  const innerHeight = height - pad.top - pad.bottom;
  const values = [...lines.values()].flat().filter((value): value is number => value !== null);
  const completed = metricKey === "completed";
  const low = completed ? Math.max(0, Math.floor((Math.min(...values, 1) - 0.02) * 10) / 10) : 0;
  const scale = completed ? null : niceTicks(Math.max(...values, 0) * 1.08 || 1, 4);
  const top = completed ? 1 : (scale?.top ?? 1);
  const ticks = completed
    ? Array.from({ length: Math.round((top - low) / 0.1) + 1 }, (_, index) =>
        Number((low + index * 0.1).toFixed(2)),
      )
    : (scale?.ticks ?? []);
  const sx = (index: number) => pad.left + (count > 1 ? index / (count - 1) : 0) * innerWidth;
  const sy = (value: number) => pad.top + (1 - (value - low) / (top - low || 1)) * innerHeight;
  const step = Math.ceil(count / X_LABELS);
  const labelled = buckets
    .map((_, index) => index)
    .filter((index) => index % step === (count - 1) % step);

  const onMove = (event: MouseEvent<HTMLDivElement>) => {
    const box = event.currentTarget.getBoundingClientRect();
    const index = Math.round(((event.clientX - box.left - pad.left) / innerWidth) * (count - 1));
    setHover(index >= 0 && index < count ? index : null);
  };
  const ranked =
    hover === null
      ? []
      : series
          .map((item) => ({ item, value: lines.get(item.key)?.[hover] ?? null }))
          .filter(
            (entry): entry is { item: AIAnalyticsSeries; value: number } => entry.value !== null,
          )
          .sort((left, right) =>
            metric.higherIsBetter ? right.value - left.value : left.value - right.value,
          );

  return (
    <AnalyticsCard
      title={t("Over time")}
      description={t("Catch a model or provider getting worse after a change.")}
      className="an-tr"
      right={
        <Seg
          className="sm"
          v={metricKey}
          opts={TREND_METRICS.map((item) => [item.value, t(item.label)] as const)}
          onChange={setMetricKey}
          label={t("Measure")}
        />
      }
    >
      <div className="an-trp" ref={ref} onMouseMove={onMove} onMouseLeave={() => setHover(null)}>
        {width > 0 && count > 0 && (
          <svg
            width={width}
            height={height}
            className="an-svg"
            role="img"
            aria-label={t("Over time")}
          >
            {ticks.map((value) => (
              <g key={value}>
                <line
                  x1={pad.left}
                  x2={pad.left + innerWidth}
                  y1={sy(value)}
                  y2={sy(value)}
                  className="an-gl"
                />
                <text x={pad.left - 10} y={sy(value) + 3.5} textAnchor="end" className="an-tk">
                  {metric.format(value)}
                </text>
              </g>
            ))}
            {labelled.map((index) => (
              <text key={index} x={sx(index)} y={height - 6} textAnchor="middle" className="an-tk">
                {bucketLabel(t, analytics, buckets[index] ?? 0, last)}
              </text>
            ))}
            {events.map(({ event, row: eventRow }) => {
              const x = sx(indexOf(event.at));
              const y = 12 + eventRow * EVENT_ROW;
              const atEnd = x > pad.left + innerWidth - EVENT_LABEL_ROOM;
              return (
                <g
                  key={`${event.provider.id ?? event.provider.name}-${event.kind}-${event.at}`}
                  className="an-ev"
                >
                  <line x1={x} x2={x} y1={y + 4} y2={pad.top + innerHeight} />
                  <circle cx={x} cy={y} r="3.5" />
                  <text x={atEnd ? x - 7 : x + 7} y={y + 3.5} textAnchor={atEnd ? "end" : "start"}>
                    {event.label}
                  </text>
                </g>
              );
            })}
            {series.map((item) => (
              <path
                key={`${item.key}-${metricKey}`}
                d={linePath(lines.get(item.key) ?? [], sx, sy)}
                pathLength={1}
                className="an-ln"
                data-series={item.key}
                style={{ "--col": colors.get(item.key) } as CSSProperties}
                {...hoverProps(setHot, item.key)}
              />
            ))}
            {hover !== null && (
              <g>
                <line
                  x1={sx(hover)}
                  x2={sx(hover)}
                  y1={pad.top}
                  y2={pad.top + innerHeight}
                  className="an-xh"
                />
                {ranked.map(({ item, value }) => (
                  <circle
                    key={item.key}
                    cx={sx(hover)}
                    cy={sy(value)}
                    r="4"
                    className="an-xd"
                    style={{ "--col": colors.get(item.key) } as CSSProperties}
                  />
                ))}
              </g>
            )}
          </svg>
        )}
        {hover !== null && ranked.length > 0 && (
          <Tip x={sx(hover)} y={pad.top} beside flip={sx(hover) > width - TIP_FLIP_ROOM}>
            <div className="an-tth">
              <b>{bucketLabel(t, analytics, buckets[hover] ?? 0, last)}</b>
            </div>
            {ranked.map(({ item, value }) => (
              <span key={item.key} className="an-ttr" data-series={item.key}>
                <em>
                  <Swatch color={colors.get(item.key)} />
                  {item.label}
                </em>
                <b className="mono">{metric.format(value)}</b>
              </span>
            ))}
          </Tip>
        )}
      </div>
    </AnalyticsCard>
  );
});
