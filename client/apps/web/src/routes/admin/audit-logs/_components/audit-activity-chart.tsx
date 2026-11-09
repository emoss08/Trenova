import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import type { AuditActivityBucket, AuditReview } from "@/lib/graphql/audit-overview";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { dateTimeFormatter, formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type PointerEvent as ReactPointerEvent } from "react";
import type { AuditActorKind } from "./audit-entry-sentence";
import {
  ACTIVITY_SERIES,
  ACTOR_KIND_LABELS,
  bucketIndexOf,
  bucketRange,
  bucketTotal,
  isOffHour,
  localHour,
  niceStep,
  shareOf,
} from "./audit-overview-model";

const HEIGHT = 176;
const PAD_LEFT = 34;
const PAD_RIGHT = 6;
const PAD_TOP = 14;
const PAD_BOTTOM = 20;
const PIN_RADIUS = 5;

type Unit = "Hour" | "Day";

type Column = { index: number; bucket: AuditActivityBucket };

function ColumnHit({
  column,
  x,
  width,
  height,
}: {
  column: Column;
  x: number;
  width: number;
  height: number;
}) {
  const hover = usePointerHoverTarget(column, String(column.bucket.at));
  return <rect x={x} y={0} width={width} height={height} fill="transparent" {...hover} />;
}

function ColumnCard({ column, unit, timezone }: { column: Column; unit: Unit; timezone: string }) {
  const t = useT();
  const at = new Date(column.bucket.at * 1000);
  const heading =
    unit === "Hour"
      ? dateTimeFormatter({ weekday: "short", hour: "numeric", timeZone: timezone }).format(at)
      : dateTimeFormatter({
          weekday: "short",
          month: "short",
          day: "numeric",
          timeZone: timezone,
        }).format(at);
  return (
    <div className="flex min-w-44 flex-col gap-1">
      <HoverCardHeading>{heading}</HoverCardHeading>
      {ACTIVITY_SERIES.map((series) => (
        <HoverCardRow
          key={series.key}
          label={ACTOR_KIND_LABELS[series.key]}
          swatchClass={series.swatch}
          value={formatNumber(column.bucket[series.key])}
        />
      ))}
      <HoverCardRow label={t("Total")} value={formatNumber(bucketTotal(column.bucket))} />
    </div>
  );
}

/**
 * Entries per hour (a day or a week) or per day (a month or more), split by who wrote them, with
 * off hours shaded and the open reviews pinned where they happened. Dragging across it narrows
 * the trail to that span; hovering the legend picks one series out.
 */
export function AuditActivityChart({
  activity,
  unit,
  timezone,
  reviews,
  range,
  onRangeChange,
  onOpenReview,
}: {
  activity: readonly AuditActivityBucket[];
  unit: Unit;
  timezone: string;
  reviews: readonly AuditReview[];
  range: { from: number; to: number } | null;
  onRangeChange: (range: { from: number; to: number }) => void;
  onOpenReview: (review: AuditReview) => void;
}) {
  const t = useT();
  const [measure, measured] = useElementWidth<HTMLDivElement>();
  const [focus, setFocus] = useState<AuditActorKind | null>(null);
  const [drag, setDrag] = useState<{ start: number; end: number } | null>(null);

  const width = Math.max(measured, 240);
  const plotWidth = width - PAD_LEFT - PAD_RIGHT;
  const plotHeight = HEIGHT - PAD_TOP - PAD_BOTTOM;
  const count = Math.max(activity.length, 1);
  const step = plotWidth / count;
  const barWidth = Math.max(step - (step > 4 ? 1 : 0.4), 0.6);
  const max = Math.max(...activity.map(bucketTotal), 0);
  const tick = niceStep(max);
  const top = Math.max(Math.ceil(max / tick) * tick, tick);
  const x = (i: number) => PAD_LEFT + i * step;
  const y = (value: number) => PAD_TOP + plotHeight * (1 - value / top);
  const totals = Object.fromEntries(
    ACTIVITY_SERIES.map((series) => [
      series.key,
      activity.reduce((sum, bucket) => sum + bucket[series.key], 0),
    ]),
  ) as Record<AuditActorKind, number>;
  const grand = activity.reduce((sum, bucket) => sum + bucketTotal(bucket), 0);

  const indexAt = (event: ReactPointerEvent<SVGSVGElement>) => {
    const box = event.currentTarget.getBoundingClientRect();
    const offset = ((event.clientX - box.left) / box.width) * width - PAD_LEFT;
    return Math.max(0, Math.min(activity.length - 1, Math.floor(offset / step)));
  };

  const selected = drag
    ? { a: Math.min(drag.start, drag.end), b: Math.max(drag.start, drag.end) }
    : range
      ? { a: bucketIndexOf(activity, range.from), b: bucketIndexOf(activity, range.to) }
      : null;

  const every = Math.ceil(activity.length / 8);
  const dayStarts = activity
    .map((bucket, index) => ({ bucket, index }))
    .filter(({ bucket, index }) =>
      unit === "Hour" ? localHour(bucket.at, timezone) === 0 : index % every === 0,
    );

  const pins = reviews
    .map((review) => ({ review, index: bucketIndexOf(activity, review.occurredAt) }))
    .filter(({ index }) => index >= 0);

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        {ACTIVITY_SERIES.map((series) => (
          <button
            key={series.key}
            type="button"
            className={cn(
              "inline-flex items-center gap-1.5 rounded-sm ui-focus-ring text-xs text-muted-foreground transition-opacity",
              focus && focus !== series.key && "opacity-45",
            )}
            onPointerEnter={() => setFocus(series.key)}
            onPointerLeave={() => setFocus(null)}
            onFocus={() => setFocus(series.key)}
            onBlur={() => setFocus(null)}
          >
            <span aria-hidden className={cn("size-2 rounded-xs", series.swatch)} />
            {ACTOR_KIND_LABELS[series.key]}
            <span className="font-medium text-foreground tabular-nums">
              {Math.round(shareOf(totals[series.key], grand) * 100)}%
            </span>
          </button>
        ))}
        <span className="ml-auto text-xs text-foreground-subtle">
          {t("Drag across the chart to narrow the trail")}
        </span>
      </div>
      <div ref={measure} className="relative min-w-0">
        <PointerHoverCard<Column>
          itemId={(column) => String(column.bucket.at)}
          render={(column) => <ColumnCard column={column} unit={unit} timezone={timezone} />}
        >
          <svg
            role="img"
            aria-label={t(
              "{0, plural, one {# entry} other {# entries}} in the window, by who wrote them",
              grand,
            )}
            width={width}
            height={HEIGHT}
            className="block touch-none select-none"
            onPointerDown={(event) => {
              if ((event.target as Element).closest("[data-pin]")) return;
              event.currentTarget.setPointerCapture(event.pointerId);
              const index = indexAt(event);
              setDrag({ start: index, end: index });
            }}
            onPointerMove={(event) => {
              if (!drag) return;
              const index = indexAt(event);
              setDrag((current) => (current ? { ...current, end: index } : current));
            }}
            onPointerUp={() => {
              if (!drag) return;
              const next = bucketRange(activity, drag.start, drag.end, unit);
              setDrag(null);
              if (next) onRangeChange(next);
            }}
            onPointerCancel={() => setDrag(null)}
          >
            {unit === "Hour"
              ? activity.map((bucket, index) =>
                  isOffHour(bucket.at, timezone) ? (
                    <rect
                      key={`off-${bucket.at}`}
                      x={x(index)}
                      y={PAD_TOP}
                      width={step}
                      height={plotHeight}
                      className="fill-muted"
                    />
                  ) : null,
                )
              : null}
            {Array.from({ length: Math.round(top / tick) + 1 }, (_, i) => i * tick).map((value) => (
              <g key={`grid-${value}`}>
                <line
                  x1={PAD_LEFT}
                  x2={width - PAD_RIGHT}
                  y1={y(value)}
                  y2={y(value)}
                  className="stroke-border"
                  strokeWidth={1}
                />
                <text
                  x={PAD_LEFT - 6}
                  y={y(value) + 3}
                  textAnchor="end"
                  className="fill-muted-foreground text-xs tabular-nums"
                >
                  {formatNumber(value)}
                </text>
              </g>
            ))}
            {dayStarts.map(({ bucket, index }) => (
              <text
                key={`label-${bucket.at}`}
                x={unit === "Hour" ? x(index) + step * 12 : x(index) + step / 2}
                y={HEIGHT - 5}
                textAnchor="middle"
                className="fill-muted-foreground text-xs"
              >
                {unit === "Hour"
                  ? t(
                      "{0} {1}",
                      dateTimeFormatter({ weekday: "short", timeZone: timezone }).format(
                        new Date(bucket.at * 1000),
                      ),
                      dateTimeFormatter({ day: "numeric", timeZone: timezone }).format(
                        new Date(bucket.at * 1000),
                      ),
                    )
                  : dateTimeFormatter({
                      month: "short",
                      day: "numeric",
                      timeZone: timezone,
                    }).format(new Date(bucket.at * 1000))}
              </text>
            ))}
            {selected && selected.a >= 0 && selected.b >= 0 ? (
              <rect
                x={x(selected.a)}
                y={PAD_TOP}
                width={x(selected.b + 1) - x(selected.a)}
                height={plotHeight}
                rx={2}
                className="fill-brand/10 stroke-brand"
                strokeWidth={1}
              />
            ) : null}
            {activity.map((bucket, index) => {
              return (
                <g key={bucket.at}>
                  {ACTIVITY_SERIES.map((series, seriesIndex) => {
                    const value = bucket[series.key];
                    if (value === 0) return null;
                    const below = ACTIVITY_SERIES.slice(0, seriesIndex).reduce(
                      (sum, earlier) => sum + bucket[earlier.key],
                      0,
                    );
                    const y0 = y(below);
                    const y1 = y(below + value);
                    return (
                      <rect
                        key={series.key}
                        x={x(index) + (step - barWidth) / 2}
                        y={y1}
                        width={barWidth}
                        height={Math.max(y0 - y1, 0.5)}
                        className={cn(
                          series.fill,
                          "transition-opacity duration-150",
                          focus && focus !== series.key && "opacity-20",
                        )}
                      />
                    );
                  })}
                  <ColumnHit
                    column={{ index, bucket }}
                    x={x(index)}
                    width={step}
                    height={HEIGHT - PAD_BOTTOM}
                  />
                </g>
              );
            })}
            {pins.map(({ review, index }) => {
              const cx = x(index) + step / 2;
              const stackTop = y(bucketTotal(activity[index]));
              const cy = Math.max(stackTop - 10, PIN_RADIUS + 1);
              return (
                <g
                  key={review.id}
                  data-pin
                  role="button"
                  tabIndex={0}
                  aria-label={review.title}
                  className="cursor-pointer ui-focus-ring"
                  onClick={() => onOpenReview(review)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onOpenReview(review);
                    }
                  }}
                >
                  <title>{review.title}</title>
                  <line
                    x1={cx}
                    x2={cx}
                    y1={cy + PIN_RADIUS}
                    y2={stackTop}
                    strokeDasharray="2 2"
                    className="stroke-warning"
                  />
                  <circle
                    cx={cx}
                    cy={cy}
                    r={PIN_RADIUS}
                    strokeWidth={1.5}
                    className="fill-background stroke-warning"
                  />
                  <circle cx={cx} cy={cy} r={2.2} className="fill-warning" />
                </g>
              );
            })}
          </svg>
        </PointerHoverCard>
      </div>
    </div>
  );
}
