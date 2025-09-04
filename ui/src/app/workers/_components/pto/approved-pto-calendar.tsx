/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { PTOCalendarEvent } from "@/services/worker";
import { useQuery } from "@tanstack/react-query";
import {
  addDays,
  endOfMonth,
  endOfWeek,
  format,
  isSameDay,
  isSameMonth,
  startOfMonth,
  startOfWeek,
} from "date-fns";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { memo, useCallback, useMemo, useState } from "react";

const PTO_COLORS = {
  Vacation:
    "dark:bg-violet-600/50 dark:border-violet-700 dark:text-violet-300 dark:hover:bg-violet-600/70 bg-violet-600 text-violet-200",
  Sick: "dark:bg-red-600/50 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-600/70 bg-red-600 text-red-200",
  Holiday:
    "dark:bg-blue-600/50 dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-600/70 bg-blue-600 text-blue-200",
  Bereavement:
    "dark:bg-emerald-600/50 dark:border-emerald-700 dark:text-emerald-300 dark:hover:bg-emerald-600/70 bg-emerald-600 text-emerald-200",
  Maternity:
    "dark:bg-pink-600/50 dark:border-pink-700 dark:text-pink-300 dark:hover:bg-pink-600/70 bg-pink-600 text-pink-200",
  Paternity:
    "dark:bg-teal-600/50 dark:border-teal-700 dark:text-teal-300 dark:hover:bg-teal-600/70 bg-teal-600 text-teal-200",
  Personal:
    "dark:bg-gray-600/50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-600/70 bg-gray-600 text-gray-200",
} as const;

const PTO_COLORS_HEX = {
  Vacation: "#7c3aed",
  Sick: "#dc2626",
  Holiday: "#2563eb",
  Bereavement: "#059669",
  Maternity: "#db2777",
  Paternity: "#0d9488",
} as const;

interface PTOCalendarProps {
  startDate: number;
  endDate: number;
  type?: string;
}

interface CalendarDay {
  date: Date;
  isCurrentMonth: boolean;
  events: PTOCalendarEvent[];
}

const EventBar = memo(function EventBar({
  event,
  dayDate,
  isHighlighted,
  onHover,
}: {
  event: PTOCalendarEvent;
  dayDate: Date;
  isHighlighted: boolean;
  onHover: (eventId: string | null) => void;
}) {
  const eventStart = new Date(event.startDate * 1000);
  const eventEnd = new Date(event.endDate * 1000);

  const dayStart = new Date(dayDate);
  dayStart.setHours(0, 0, 0, 0);
  const dayEnd = new Date(dayDate);
  dayEnd.setHours(23, 59, 59, 999);

  const isStart = eventStart >= dayStart && eventStart <= dayEnd;
  const isEnd = eventEnd >= dayStart && eventEnd <= dayEnd;

  const eventColorClasses =
    PTO_COLORS[event.type as keyof typeof PTO_COLORS] ||
    "bg-gray-500 border-gray-600 text-white hover:bg-gray-600";

  const barClassName = cn(
    "relative text-[11px] px-2 font-semibold cursor-pointer transition-all duration-150",
    "min-h-[22px] flex items-center border",
    eventColorClasses,
    isStart && "rounded-l-md ml-0.5",
    isEnd && "rounded-r-md mr-0.5",
    !isStart && "rounded-l-none border-l ml-0",
    !isEnd && "rounded-r-none border-r mr-0",
    isHighlighted ? "z-20 shadow-lg ring-1 ring-inset" : "hover:z-10",
  );

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div
            className={barClassName}
            onMouseEnter={() => onHover(event.id)}
            onMouseLeave={() => onHover(null)}
          >
            {isStart ? (
              <span className="truncate text-[10px] leading-none">
                {event.workerName}
              </span>
            ) : (
              <span className="w-full h-full" />
            )}
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <div className="text-xs space-y-1">
            <p className="font-medium">{event.workerName}</p>
            <p>{event.type}</p>
            <p>
              {format(eventStart, "MMM d, yyyy")} -{" "}
              {format(eventEnd, "MMM d, yyyy")}
            </p>
            {event.reason && (
              <p className="text-muted-foreground">{event.reason}</p>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
});

const CalendarGrid = memo(function CalendarGrid({
  days,
  highlightedEventId,
  onEventHover,
}: {
  days: CalendarDay[];
  highlightedEventId: string | null;
  onEventHover: (eventId: string | null) => void;
}) {
  const weekDays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

  return (
    <div className="border border-border rounded-lg overflow-hidden bg-background">
      <div className="grid grid-cols-7 border-b border-border">
        {weekDays.map((day) => (
          <div
            key={day}
            className="px-2 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground text-center border-r border-border last:border-r-0"
          >
            {day}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7">
        {days.map((day, index) => (
          <div
            key={index}
            className={cn(
              "min-h-[100px] border-r border-b border-border last:border-r-0",
              "hover:bg-accent/5 transition-colors",
              !day.isCurrentMonth && "bg-muted/5",
              index >= days.length - 7 && "border-b-0",
            )}
          >
            <div className="p-2">
              <div className="flex items-start justify-between mb-1">
                <span
                  className={cn(
                    "text-sm leading-none",
                    isSameDay(day.date, new Date())
                      ? "bg-primary text-primary-foreground rounded-full size-6 flex items-center justify-center font-semibold"
                      : "text-foreground",
                    !day.isCurrentMonth && "text-muted-foreground",
                  )}
                >
                  {format(day.date, "d")}
                </span>
              </div>
              <div className="space-y-0.5">
                {day.events.slice(0, 3).map((event) => (
                  <EventBar
                    key={event.id}
                    event={event}
                    dayDate={day.date}
                    isHighlighted={highlightedEventId === event.id}
                    onHover={onEventHover}
                  />
                ))}
                {day.events.length > 3 && (
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button className="text-[10px] font-medium text-muted-foreground hover:text-foreground transition-colors px-2 py-0.5">
                          +{day.events.length - 3} more
                        </button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <div className="text-xs space-y-2 max-h-[300px] overflow-y-auto">
                          {day.events.slice(3).map((event) => {
                            const colorHex =
                              PTO_COLORS_HEX[
                                event.type as keyof typeof PTO_COLORS_HEX
                              ] || "#6b7280";
                            return (
                              <div
                                key={event.id}
                                className="flex items-center gap-2"
                              >
                                <div
                                  className="w-2 h-2 rounded-full shrink-0"
                                  style={{ backgroundColor: colorHex }}
                                />
                                <div>
                                  <p className="font-medium">
                                    {event.workerName}
                                  </p>
                                  <p className="text-muted-foreground text-[11px]">
                                    {event.type}
                                  </p>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
});

export default function PTOCalendar({ type }: PTOCalendarProps) {
  const [currentMonth, setCurrentMonth] = useState(() => {
    const today = new Date();
    return new Date(today.getFullYear(), today.getMonth(), 1);
  });
  const [highlightedEventId, setHighlightedEventId] = useState<string | null>(
    null,
  );

  const monthStart = useMemo(() => startOfMonth(currentMonth), [currentMonth]);
  const monthEnd = useMemo(() => endOfMonth(currentMonth), [currentMonth]);

  const calendarStart = useMemo(
    () => startOfWeek(monthStart, { weekStartsOn: 0 }),
    [monthStart],
  );
  const calendarEnd = useMemo(
    () => endOfWeek(monthEnd, { weekStartsOn: 0 }),
    [monthEnd],
  );

  const calendarStartTimestamp = useMemo(
    () => Math.floor(calendarStart.getTime() / 1000),
    [calendarStart],
  );
  const calendarEndTimestamp = useMemo(
    () => Math.floor(calendarEnd.getTime() / 1000),
    [calendarEnd],
  );

  const query = useQuery({
    ...queries.worker.getPTOCalendarData({
      startDate: calendarStartTimestamp,
      endDate: calendarEndTimestamp,
      type: type || undefined,
    }),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: Boolean(calendarStartTimestamp && calendarEndTimestamp),
  });

  const calendarDays = useMemo((): CalendarDay[] => {
    const days: CalendarDay[] = [];
    let currentDate = new Date(calendarStart);

    while (currentDate <= calendarEnd) {
      const currentDateCopy = new Date(currentDate);
      const dayEvents =
        query.data?.filter((event) => {
          const eventStart = new Date(event.startDate * 1000);
          const eventEnd = new Date(event.endDate * 1000);

          // Check if the current day falls within the event's date range
          const dayStart = new Date(currentDateCopy);
          dayStart.setHours(0, 0, 0, 0);
          const dayEnd = new Date(currentDateCopy);
          dayEnd.setHours(23, 59, 59, 999);

          // Event overlaps with this day if:
          // - Event starts before or on this day AND ends after or on this day
          const overlaps = eventStart <= dayEnd && eventEnd >= dayStart;

          return overlaps;
        }) || [];

      days.push({
        date: new Date(currentDate),
        isCurrentMonth: isSameMonth(currentDate, currentMonth),
        events: dayEvents,
      });

      currentDate = addDays(currentDate, 1);
    }

    return days;
  }, [calendarStart, calendarEnd, currentMonth, query.data]);

  const handlePreviousMonth = useCallback(() => {
    setCurrentMonth(
      (prev) => new Date(prev.getFullYear(), prev.getMonth() - 1),
    );
  }, []);

  const handleNextMonth = useCallback(() => {
    setCurrentMonth(
      (prev) => new Date(prev.getFullYear(), prev.getMonth() + 1),
    );
  }, []);

  const handleToday = useCallback(() => {
    const today = new Date();
    const todayTimestamp = Math.floor(today.getTime() / 1000);

    if (query.data && query.data.length > 0) {
      const hasTodayPTO = query.data.some(
        (event) =>
          event.startDate <= todayTimestamp && event.endDate >= todayTimestamp,
      );

      if (hasTodayPTO) {
        setCurrentMonth(today);
      } else {
        const firstPTO = query.data.reduce((earliest, current) =>
          current.startDate < earliest.startDate ? current : earliest,
        );
        setCurrentMonth(new Date(firstPTO.startDate * 1000));
      }
    } else {
      setCurrentMonth(today);
    }
  }, [query.data]);

  if (query.isLoading) {
    return <Skeleton className="h-[500px] w-full" />;
  }

  if (query.isError) {
    return (
      <div className="h-[500px] w-full flex items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-destructive mb-2">
            Failed to load calendar data
          </p>
          <p className="text-xs text-muted-foreground">
            {query.error?.message || "An error occurred"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={handlePreviousMonth}
            className="size-8 hover:bg-accent"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleNextMonth}
            className="size-8 hover:bg-accent"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <h3 className="text-lg font-semibold ml-2">
            {format(currentMonth, "MMMM yyyy")}
          </h3>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleToday}
          className="text-xs"
        >
          Today
        </Button>
      </div>

      <CalendarGrid
        days={calendarDays}
        highlightedEventId={highlightedEventId}
        onEventHover={setHighlightedEventId}
      />

      <div className="flex flex-wrap gap-3 text-xs">
        {Object.entries(PTO_COLORS).map(([type, classes]) => {
          const baseClasses = classes.split(" ").slice(0, 2).join(" ");
          return (
            <div key={type} className="flex items-center gap-2">
              <div className={cn("size-3 rounded-sm border", baseClasses)} />
              <span className="text-muted-foreground font-medium">{type}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
