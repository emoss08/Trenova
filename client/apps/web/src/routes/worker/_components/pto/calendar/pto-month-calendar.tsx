import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { fetchUpcomingWorkerPTO } from "@/lib/queries/worker";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRange, getTodayDate } from "@trenova/shared/lib/date";
import { PTO_STATUS_BAR_CLASS, ptoTypeMeta } from "@trenova/shared/lib/pto";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { PTOFilter, PTOType, WorkerPTO } from "@trenova/shared/types/worker";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PTOStatusBadge } from "@trenova/shared/components/status-badge";
import { ptoDaysOf } from "../pto-columns";
import { PTOFormDialog } from "../pto-form-dialog";
import { UpcomingPTOContent } from "../requested/upcoming-pto-content";
import {
  buildMonthGrid,
  buildWeekSegments,
  isWithin,
  laneCount,
  monthEndUnix,
  monthOf,
  monthStartUnix,
  selectionRange,
  shiftMonth,
  type CalendarDay,
  type CalendarSegment,
} from "./calendar-layout";
import { WhosOutStrip } from "./whos-out-strip";

const CALENDAR_PAGE_SIZE = 200;
const LANE_HEIGHT = 22;
const LANE_GAP = 2;
const HEADER_HEIGHT = 24;
const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTH_FORMAT = new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" });

type SelectionRange = { start: number; end: number };

export type PTOMonthCalendarProps = {
  filters: PTOFilter;
  onMonthChange: (startDate: number, endDate: number) => void;
};

export function PTOMonthCalendar({ filters, onMonthChange }: PTOMonthCalendarProps) {
  const user = useAuthStore((state) => state.user);
  const { allowed: canCreate } = usePermission(Resource.WorkerPTO, Operation.Create);
  const { year, month } = monthOf(filters.startDate);
  const rangeStart = monthStartUnix(year, month);
  const rangeEnd = monthEndUnix(year, month);
  const todayUnix = getTodayDate();

  const query = useQuery({
    queryKey: [
      ...queries.worker.listUpcomingPTO._def,
      "calendar",
      {
        startDate: rangeStart,
        endDate: rangeEnd,
        type: filters.type,
        workerId: filters.workerId,
        fleetCodeId: filters.fleetCodeId,
        timezone: user?.timezone,
      },
    ],
    queryFn: ({ signal }) =>
      fetchUpcomingWorkerPTO(
        {
          filter: { limit: CALENDAR_PAGE_SIZE },
          type: filters.type as PTOType | undefined,
          startDate: rangeStart,
          endDate: rangeEnd,
          workerId: filters.workerId,
          fleetCodeId: filters.fleetCodeId,
          timezone: user?.timezone,
        },
        { signal },
      ),
    staleTime: 60 * 1000,
  });

  const items = useMemo(
    () =>
      (query.data?.results ?? []).filter(
        (pto) => pto.status === "Approved" || pto.status === "Requested",
      ),
    [query.data],
  );
  const weeks = useMemo(() => buildMonthGrid(year, month), [year, month]);

  const [anchor, setAnchor] = useState<number | null>(null);
  const [focus, setFocus] = useState<number | null>(null);
  const [requestRange, setRequestRange] = useState<SelectionRange | null>(null);
  const dragging = useRef(false);

  const selection = useMemo<SelectionRange | null>(
    () => (anchor !== null && focus !== null ? selectionRange(anchor, focus) : null),
    [anchor, focus],
  );

  const finishSelection = useCallback(() => {
    if (!dragging.current) return;
    dragging.current = false;
    if (anchor !== null && focus !== null && canCreate) {
      setRequestRange(selectionRange(anchor, focus));
    }
    setAnchor(null);
    setFocus(null);
  }, [anchor, canCreate, focus]);

  useEffect(() => {
    window.addEventListener("mouseup", finishSelection);
    return () => window.removeEventListener("mouseup", finishSelection);
  }, [finishSelection]);

  const goToMonth = useCallback(
    (delta: number) => {
      const next = shiftMonth(year, month, delta);
      onMonthChange(monthStartUnix(next.year, next.month), monthEndUnix(next.year, next.month));
    },
    [month, onMonthChange, year],
  );

  const onKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      goToMonth(-1);
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      goToMonth(1);
    } else if (event.key === "Escape") {
      dragging.current = false;
      setAnchor(null);
      setFocus(null);
    }
  };

  return (
    <div
      className="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden"
      data-testid="pto-month-calendar"
    >
      <WhosOutStrip items={items} todayUnix={todayUnix} />
      <div className="flex shrink-0 items-center justify-between">
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            className="size-7"
            aria-label="Previous month"
            onClick={() => goToMonth(-1)}
          >
            <ChevronLeftIcon className="size-4" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="size-7"
            aria-label="Next month"
            onClick={() => goToMonth(1)}
          >
            <ChevronRightIcon className="size-4" />
          </Button>
          <h4 className="ml-1 text-sm font-semibold">
            {MONTH_FORMAT.format(new Date(year, month, 1))}
          </h4>
        </div>
        <p className="text-muted-foreground text-[11px]">
          {canCreate ? "Drag across days to request time off · " : ""}
          {items.length} on the calendar
        </p>
      </div>

      {query.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : query.isError ? (
        <p className="text-destructive text-xs">Could not load the calendar. Try again shortly.</p>
      ) : (
        <div
          role="grid"
          aria-label="PTO calendar"
          tabIndex={0}
          onKeyDown={onKeyDown}
          className="focus-visible:ring-ring flex min-h-0 flex-1 select-none flex-col overflow-auto rounded-md border focus-visible:ring-2 focus-visible:outline-none"
          data-testid="pto-calendar-grid"
        >
          <div className="bg-muted grid shrink-0 grid-cols-7 border-b text-[11px] font-medium sticky top-0 z-20">
            {WEEKDAYS.map((weekday) => (
              <div key={weekday} className="text-muted-foreground px-2 py-1">
                {weekday}
              </div>
            ))}
          </div>
          {weeks.map((week) => {
            const segments = buildWeekSegments(items, week);
            const lanes = laneCount(segments);
            const height = HEADER_HEIGHT + Math.max(lanes, 1) * (LANE_HEIGHT + LANE_GAP) + 4;
            return (
              <div
                key={week.key}
                role="row"
                className="relative grid shrink-0 grid-cols-7 border-b last:border-b-0"
                style={{ minHeight: height }}
              >
                {week.days.map((day) => (
                  <DayCell
                    key={day.key}
                    day={day}
                    selected={isWithin(day.unix, selection)}
                    onMouseDown={() => {
                      if (!canCreate) return;
                      dragging.current = true;
                      setAnchor(day.unix);
                      setFocus(day.unix);
                    }}
                    onMouseEnter={() => {
                      if (dragging.current) setFocus(day.unix);
                    }}
                  />
                ))}
                {segments.map((segment) => (
                  <SpanBar key={`${segment.item.id}-${week.key}`} segment={segment} />
                ))}
              </div>
            );
          })}
        </div>
      )}

      {requestRange ? (
        <PTOFormDialog
          open
          onOpenChange={(open) => {
            if (!open) setRequestRange(null);
          }}
          defaultRange={requestRange}
        />
      ) : null}
    </div>
  );
}

function DayCell({
  day,
  selected,
  onMouseDown,
  onMouseEnter,
}: {
  day: CalendarDay;
  selected: boolean;
  onMouseDown: () => void;
  onMouseEnter: () => void;
}) {
  return (
    <div
      role="gridcell"
      data-testid={`pto-day-${day.key}`}
      data-selected={selected ? "true" : undefined}
      aria-selected={selected}
      onMouseDown={onMouseDown}
      onMouseEnter={onMouseEnter}
      className={cn(
        "border-r px-1.5 py-1 text-[11px] last:border-r-0",
        !day.inMonth && "text-muted-foreground/60 bg-muted/20",
        day.isWeekend && day.inMonth && "bg-muted/10",
        selected && "bg-primary/10",
      )}
    >
      <span
        className={cn(
          "inline-flex size-5 items-center justify-center rounded-full tabular-nums",
          day.isToday && "bg-primary text-primary-foreground font-semibold",
        )}
      >
        {day.date}
      </span>
    </div>
  );
}

function SpanBar({ segment }: { segment: CalendarSegment<WorkerPTO> }) {
  const pto = segment.item;
  const meta = ptoTypeMeta(pto.type);
  const left = `${(segment.startCol / 7) * 100}%`;
  const width = `${((segment.endCol - segment.startCol + 1) / 7) * 100}%`;
  const top = HEADER_HEIGHT + segment.lane * (LANE_HEIGHT + LANE_GAP);
  const name = `${pto.worker?.firstName ?? ""} ${pto.worker?.lastName ?? ""}`.trim();

  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            data-testid={`pto-span-${pto.id}`}
            title={`${name} · ${meta.label} · ${formatRange(pto.startDate, pto.endDate)}`}
            className={cn(
              "absolute z-10 flex h-[22px] items-center truncate px-1.5 text-[11px] font-medium",
              meta.barClass,
              PTO_STATUS_BAR_CLASS[pto.status],
              segment.continuesBefore ? "rounded-l-none" : "rounded-l-md",
              segment.continuesAfter ? "rounded-r-none" : "rounded-r-md",
            )}
            style={{ left: `calc(${left} + 2px)`, width: `calc(${width} - 4px)`, top }}
            onMouseDown={(event) => event.stopPropagation()}
          />
        }
      >
        {name || meta.label}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80">
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between gap-2">
            <PTOStatusBadge status={pto.status} />
            <span className="text-muted-foreground text-xs tabular-nums">
              {ptoDaysOf(pto)} day{ptoDaysOf(pto) === 1 ? "" : "s"}
            </span>
          </div>
          <UpcomingPTOContent pto={pto} />
          {pto.reason ? <p className="text-muted-foreground text-xs">{pto.reason}</p> : null}
        </div>
      </PopoverContent>
    </Popover>
  );
}
