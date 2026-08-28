import { cn } from "@trenova/shared/lib/utils";
import { DAY_LABEL_HEIGHT_PX, HOUR_TICK_HEIGHT_PX, RAIL_WIDTH_PX } from "./constants";
import { secondsToX, type DayColumn, type HourTick, type TimeRange } from "./time-scale";
import type { TimelineZoom } from "../url-state";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";

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
    <div className="border-border sticky top-0 z-40 flex border-b" style={{ height: headerHeight }}>
      <div
        className="border-border bg-muted sticky left-0 z-50 flex shrink-0 items-center border-r px-2.5"
        style={{ width: RAIL_WIDTH_PX }}
      >
        <span className="text-muted-foreground text-[9.5px] font-semibold tracking-wide uppercase">
          Drivers · {driverCount}
        </span>
      </div>
      <div className="bg-muted relative shrink-0" style={{ width: canvasWidth }}>
        {dayColumns.map((day) => (
          <div
            key={day.start}
            className={cn(
              "border-border/70 absolute top-0 flex items-center border-l px-1.5 first:border-l-0",
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
              className="font-table text-muted-foreground absolute -translate-x-1/2 text-[9px] tabular-nums"
              style={{ left: tick.x, top: DAY_LABEL_HEIGHT_PX + 3 }}
            >
              {tick.label}
            </span>
          ))}
        {nowInRange && (
          <span
            className="bg-brand font-table text-brand-foreground absolute bottom-0 z-10 -translate-x-1/2 rounded-t px-1 py-px text-[8.5px] font-semibold tabular-nums"
            style={{ left: nowX }}
          >
            {formatUnixInUserTimezone(now, { hour: "2-digit", minute: "2-digit", hour12: false })}
          </span>
        )}
      </div>
    </div>
  );
}
