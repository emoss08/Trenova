import { cn } from "@trenova/shared/lib/utils";
import { format } from "date-fns";
import {
  DAY_LABEL_HEIGHT_PX,
  HOUR_TICK_HEIGHT_PX,
  RAIL_WIDTH_PX,
} from "./constants";
import { secondsToX, type DayColumn, type HourTick, type TimeRange } from "./time-scale";
import type { TimelineZoom } from "../url-state";

type TimelineHeaderProps = {
  range: TimeRange;
  zoom: TimelineZoom;
  canvasWidth: number;
  dayColumns: DayColumn[];
  hourTicks: HourTick[];
  now: number;
  driverCount: number;
};

export function TimelineHeader({
  range,
  zoom,
  canvasWidth,
  dayColumns,
  hourTicks,
  now,
  driverCount,
}: TimelineHeaderProps) {
  const hasHourRow = hourTicks.length > 0;
  const headerHeight = DAY_LABEL_HEIGHT_PX + (hasHourRow ? HOUR_TICK_HEIGHT_PX : 0);
  const nowInRange = now >= range.start && now < range.end;
  const nowX = secondsToX(now, range, zoom);

  return (
    <div className="sticky top-0 z-40 flex border-b border-border" style={{ height: headerHeight }}>
      <div
        className="sticky left-0 z-50 flex shrink-0 items-center border-r border-border bg-muted px-2.5"
        style={{ width: RAIL_WIDTH_PX }}
      >
        <span className="text-[9.5px] font-semibold tracking-wide text-muted-foreground uppercase">
          Drivers · {driverCount}
        </span>
      </div>
      <div className="relative shrink-0 bg-muted" style={{ width: canvasWidth }}>
        {dayColumns.map((day) => (
          <div
            key={day.start}
            className={cn(
              "absolute top-0 flex items-center border-l border-border/70 px-1.5 first:border-l-0",
              day.isToday && "text-brand",
            )}
            style={{ left: day.x, width: day.width, height: DAY_LABEL_HEIGHT_PX }}
          >
            <span className="truncate text-[10px] font-semibold tracking-wide uppercase">
              {day.label}
            </span>
          </div>
        ))}
        {hasHourRow &&
          hourTicks.map((tick) => (
            <span
              key={tick.time}
              className="absolute -translate-x-1/2 font-table text-[9px] text-muted-foreground tabular-nums"
              style={{ left: tick.x, top: DAY_LABEL_HEIGHT_PX + 3 }}
            >
              {tick.label}
            </span>
          ))}
        {nowInRange && (
          <span
            className="absolute bottom-0 z-10 -translate-x-1/2 rounded-t bg-brand px-1 py-px font-table text-[8.5px] font-semibold text-brand-foreground tabular-nums"
            style={{ left: nowX }}
          >
            {format(new Date(now * 1000), "HH:mm")}
          </span>
        )}
      </div>
    </div>
  );
}
