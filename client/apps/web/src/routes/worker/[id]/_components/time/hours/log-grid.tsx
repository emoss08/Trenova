import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { dutyStatusLabel } from "@/lib/hos";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useId, useRef } from "react";
import {
  formatLaneTotal,
  laneCenterY,
  LOG_LANE_CODES,
  logHourLabels,
  logPath,
  logX,
  type LaneTotals,
  type LogGridGeometry,
  type LogLane,
  type LogSegment,
} from "./hos-model";
import { formatHoursMinutes } from "./hos-format";

const MIN_WIDTH = 760;
const LEFT = 40;
const RIGHT = 64;
const TOP = 22;
const LANE_HEIGHT = 30;
const BOTTOM = 20;
const GRID_HEIGHT = LANE_HEIGHT * 4;
const HEIGHT = TOP + GRID_HEIGHT + BOTTOM;
const QUARTERS = [15, 30, 45] as const;
const DRAW_MS = 420;
const SETTLE = "cubic-bezier(0.16, 1, 0.3, 1)";

function prefersReducedMotion(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}

export type LogGridProps = {
  segments: readonly LogSegment[];
  dayStart: number;
  dayEnd: number;
  totals: LaneTotals;
  /** Today's current time, which stops the line with a "now" tick; null for a finished day. */
  nowSec: number | null;
  /** The day in words, for the graph's accessible name. */
  dayLabel: string;
  /** Changing it draws the line again: the selected day. */
  drawKey: string;
};

/**
 * The day's log as a paper log draws it: 24 hour columns with quarter-hour ticks, four lanes
 * (off duty, sleeper berth, driving, on duty), a stepped line through the day's statuses and each
 * lane's total at the right. Narrower than the paper, it scrolls inside its own box.
 */
export function LogGrid({
  segments,
  dayStart,
  dayEnd,
  totals,
  nowSec,
  dayLabel,
  drawKey,
}: LogGridProps) {
  const t = useT();
  const [measureRef, measured] = useElementWidth<HTMLDivElement>();
  const width = Math.max(MIN_WIDTH, measured);
  const geometry: LogGridGeometry = {
    left: LEFT,
    width: width - LEFT - RIGHT,
    top: TOP,
    laneHeight: LANE_HEIGHT,
  };
  const hourWidth = geometry.width / 24;
  const x = (sec: number) => logX(sec, dayStart, dayEnd, geometry);
  const hatchId = `log-hatch-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const pathRef = useRef<SVGPathElement>(null);
  const drawnKey = useRef<string | null>(null);
  const path = logPath(segments, dayStart, dayEnd, geometry);
  const sum = totals[0] + totals[1] + totals[2] + totals[3];
  const labels = logHourLabels(t("M"), t("N"));
  const nowX = nowSec !== null && nowSec > dayStart && nowSec < dayEnd ? x(nowSec) : null;

  useEffect(() => {
    const element = pathRef.current;
    if (!element || drawnKey.current === drawKey) return;
    drawnKey.current = drawKey;
    if (prefersReducedMotion() || typeof element.getTotalLength !== "function") return;
    const length = Math.ceil(element.getTotalLength());
    if (length === 0 || typeof element.animate !== "function") return;
    element.style.strokeDasharray = String(length);
    const animation = element.animate([{ strokeDashoffset: length }, { strokeDashoffset: 0 }], {
      duration: DRAW_MS,
      easing: SETTLE,
    });
    const settle = () => {
      element.style.strokeDasharray = "";
    };
    animation.onfinish = settle;
    animation.oncancel = settle;
    return () => animation.cancel();
  }, [drawKey, path]);

  return (
    <div ref={measureRef} className="w-full min-w-0">
      <div className="[scrollbar-width:thin] overflow-x-auto overflow-y-hidden">
        <PointerHoverCard<LogSegment>
          itemId={(segment) => `${drawKey}-${segment.index}`}
          render={(segment) => <SegmentCard segment={segment} />}
        >
          <svg
            width={width}
            height={HEIGHT}
            viewBox={`0 0 ${width} ${HEIGHT}`}
            role="img"
            aria-label={t(
              "Duty status graph, {0}. Off duty {1}, sleeper berth {2}, driving {3}, on duty {4}.",
              dayLabel,
              formatLaneTotal(totals[0]),
              formatLaneTotal(totals[1]),
              formatLaneTotal(totals[2]),
              formatLaneTotal(totals[3]),
            )}
            className="block overflow-visible"
          >
            <defs>
              <pattern
                id={hatchId}
                width={5}
                height={5}
                patternUnits="userSpaceOnUse"
                patternTransform="rotate(45)"
              >
                <line x1={0} y1={0} x2={0} y2={5} className="stroke-border-subtle" />
              </pattern>
            </defs>
            {labels.map((label, hour) => (
              <text
                key={hour}
                x={LEFT + hour * hourWidth}
                y={14}
                textAnchor="middle"
                className={cn(
                  "text-2xs font-medium tabular-nums",
                  hour % 12 === 0 ? "fill-foreground" : "fill-foreground-subtle",
                )}
              >
                {label}
              </text>
            ))}
            <text
              x={width - 8}
              y={14}
              textAnchor="end"
              className="fill-foreground-subtle text-2xs font-medium"
            >
              {t("Total")}
            </text>
            {nowX !== null ? (
              <rect
                x={nowX}
                y={TOP}
                width={LEFT + geometry.width - nowX}
                height={GRID_HEIGHT}
                fill={`url(#${hatchId})`}
              />
            ) : null}
            {Array.from({ length: 23 }, (_, index) => index + 1).map((hour) => (
              <line
                key={hour}
                x1={LEFT + hour * hourWidth}
                x2={LEFT + hour * hourWidth}
                y1={TOP}
                y2={TOP + GRID_HEIGHT}
                shapeRendering="crispEdges"
                className="stroke-border"
              />
            ))}
            {LOG_LANE_CODES.map((code, laneIndex) => {
              const lane = laneIndex as LogLane;
              const top = TOP + lane * LANE_HEIGHT;
              return (
                <g key={code}>
                  {Array.from({ length: 24 }, (_, hour) =>
                    QUARTERS.map((quarter) => {
                      const tickX = LEFT + hour * hourWidth + (quarter / 60) * hourWidth;
                      return (
                        <line
                          key={`${hour}-${quarter}`}
                          x1={tickX}
                          x2={tickX}
                          y1={top}
                          y2={top + (quarter === 30 ? 9 : 5)}
                          shapeRendering="crispEdges"
                          className="stroke-border-strong"
                        />
                      );
                    }),
                  )}
                  {lane > 0 ? (
                    <line
                      x1={LEFT}
                      x2={LEFT + geometry.width}
                      y1={top}
                      y2={top}
                      shapeRendering="crispEdges"
                      className="stroke-border-strong"
                    />
                  ) : null}
                  <text
                    x={LEFT - 8}
                    y={laneCenterY(lane, geometry) + 3.5}
                    textAnchor="end"
                    className="fill-muted-foreground font-mono text-2xs font-medium"
                  >
                    {code}
                  </text>
                  <text
                    x={width - 8}
                    y={laneCenterY(lane, geometry) + 4}
                    textAnchor="end"
                    className={cn(
                      "font-mono text-xs font-medium tabular-nums",
                      totals[lane] > 0 ? "fill-foreground" : "fill-foreground-faint",
                    )}
                  >
                    {formatLaneTotal(totals[lane])}
                  </text>
                </g>
              );
            })}
            <rect
              x={LEFT + 0.5}
              y={TOP + 0.5}
              width={geometry.width - 1}
              height={GRID_HEIGHT - 1}
              fill="none"
              shapeRendering="crispEdges"
              className="stroke-border-strong"
            />
            <text
              x={width - 8}
              y={TOP + GRID_HEIGHT + 14}
              textAnchor="end"
              className="fill-muted-foreground font-mono text-2xs"
            >
              {formatLaneTotal(sum)}
            </text>
            {segments.map((segment) => (
              <SegmentHit
                key={segment.index}
                segment={segment}
                id={`${drawKey}-${segment.index}`}
                x={x(segment.startSec)}
                width={Math.max(1, x(segment.endSec) - x(segment.startSec))}
                y={TOP + segment.lane * LANE_HEIGHT + 1}
                height={LANE_HEIGHT - 1}
                label={segmentLabel(t, segment)}
              />
            ))}
            {path ? (
              <path
                ref={pathRef}
                d={path}
                fill="none"
                strokeWidth={2}
                strokeLinejoin="miter"
                strokeLinecap="square"
                pointerEvents="none"
                className="stroke-foreground"
              />
            ) : null}
            {nowX !== null && nowSec !== null ? (
              <>
                <line
                  x1={nowX}
                  x2={nowX}
                  y1={TOP - 4}
                  y2={TOP + GRID_HEIGHT + 5}
                  shapeRendering="crispEdges"
                  className="stroke-foreground"
                />
                <text
                  x={nowX + 4}
                  y={TOP + GRID_HEIGHT + 14}
                  className="fill-foreground text-2xs font-medium"
                >
                  {t("now, {0}", formatUnixTime(nowSec))}
                </text>
              </>
            ) : null}
          </svg>
        </PointerHoverCard>
      </div>
    </div>
  );
}

function segmentLabel(t: TranslateFn, segment: LogSegment): string {
  const end = segment.ongoing ? t("now") : formatUnixTime(segment.endSec);
  const base = t(
    "{0}, {1} to {2}",
    dutyStatusLabel(segment.status),
    formatUnixTime(segment.startSec),
    end,
  );
  return segment.remark ? t("{0}. {1}", base, segment.remark) : base;
}

function SegmentHit({
  segment,
  id,
  x,
  width,
  y,
  height,
  label,
}: {
  segment: LogSegment;
  id: string;
  x: number;
  width: number;
  y: number;
  height: number;
  label: string;
}) {
  const hover = usePointerHoverTarget(segment, id);
  return (
    <rect
      {...hover}
      x={x}
      y={y}
      width={width}
      height={height}
      tabIndex={0}
      aria-label={label}
      className="fill-transparent outline-none hover:fill-surface-hover focus-visible:fill-brand-subtle"
    />
  );
}

function SegmentCard({ segment }: { segment: LogSegment }) {
  const t = useT();
  return (
    <>
      <HoverCardHeading>{dutyStatusLabel(segment.status)}</HoverCardHeading>
      <p className="text-muted-foreground tabular-nums">
        {t(
          "{0} to {1} · {2}",
          formatUnixTime(segment.startSec),
          segment.ongoing ? t("now") : formatUnixTime(segment.endSec),
          formatHoursMinutes(t, (segment.endSec - segment.startSec) * 1000),
        )}
      </p>
      {segment.vehicleName ? (
        <HoverCardRow label={t("Vehicle")} value={segment.vehicleName} />
      ) : null}
      {segment.remark ? <p className="max-w-64 text-muted-foreground">{segment.remark}</p> : null}
    </>
  );
}
