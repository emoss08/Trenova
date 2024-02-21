/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  faChevronLeft,
  faChevronRight,
} from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  CalendarDate,
  isToday as _isToday,
  createCalendar,
  getLocalTimeZone,
  getWeeksInMonth,
} from "@internationalized/date";
import React, { useMemo } from "react";
import {
  CalendarProps,
  DateValue,
  useButton,
  useCalendar,
  useCalendarCell,
  useCalendarGrid,
  useLocale,
} from "react-aria";
import { CalendarState, useCalendarState } from "react-stately";

function Calendar(props: CalendarProps<DateValue>) {
  const prevButtonRef = React.useRef<HTMLButtonElement | null>(null);
  const nextButtonRef = React.useRef<HTMLButtonElement | null>(null);

  const { locale } = useLocale();
  const state = useCalendarState({
    ...props,
    locale,
    createCalendar,
  });
  const {
    calendarProps,
    prevButtonProps: _prevButtonProps,
    nextButtonProps: _nextButtonProps,
    title,
  } = useCalendar(props, state);
  const { buttonProps: prevButtonProps } = useButton(
    _prevButtonProps,
    prevButtonRef,
  );
  const { buttonProps: nextButtonProps } = useButton(
    _nextButtonProps,
    nextButtonRef,
  );

  return (
    <div {...calendarProps} className="space-y-4">
      <div className="relative flex items-center justify-center pt-1">
        <Button
          {...prevButtonProps}
          ref={prevButtonRef}
          variant={"outline"}
          className={cn(
            "absolute left-1 h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100",
          )}
        >
          <FontAwesomeIcon icon={faChevronLeft} className="size-4" />
        </Button>
        <div className="text-sm font-medium">{title}</div>
        <Button
          {...nextButtonProps}
          ref={nextButtonRef}
          variant={"outline"}
          className={cn(
            "absolute right-1 h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100",
          )}
        >
          <FontAwesomeIcon icon={faChevronRight} className="size-4" />
        </Button>
      </div>
      <CalendarGrid state={state} />
    </div>
  );
}

interface CalendarGridProps {
  state: CalendarState;
}

function CalendarGrid({ state, ...props }: CalendarGridProps) {
  const { locale } = useLocale();
  const { gridProps, headerProps, weekDays } = useCalendarGrid(props, state);

  // Get the number of weeks in the month so we can render the proper number of rows.
  const weeksInMonth = getWeeksInMonth(state.visibleRange.start, locale);

  return (
    <table
      {...gridProps}
      className={cn(gridProps.className, "w-full border-collapse space-y-1")}
    >
      <thead {...headerProps}>
        <tr className="flex">
          {weekDays.map((day, index) => (
            <th
              className="w-9 rounded-md text-[0.8rem] font-normal text-muted-foreground"
              key={index}
            >
              {day}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {[...new Array(weeksInMonth).keys()].map((weekIndex) => (
          <tr className="mt-2 flex w-full" key={weekIndex}>
            {state
              .getDatesInWeek(weekIndex)
              .map((date, i) =>
                date ? (
                  <CalendarCell key={i} state={state} date={date} />
                ) : (
                  <td key={i} />
                ),
              )}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

interface CalendarCellProps {
  state: CalendarState;
  date: CalendarDate;
}

function CalendarCell({ state, date }: CalendarCellProps) {
  const ref = React.useRef<HTMLButtonElement | null>(null);
  const {
    cellProps,
    buttonProps,
    isSelected,
    isOutsideVisibleRange,
    isDisabled,
    formattedDate,
  } = useCalendarCell({ date }, state, ref);

  const isToday = useMemo(() => {
    const timezone = getLocalTimeZone();
    return _isToday(date, timezone);
  }, [date]);

  return (
    <td
      {...cellProps}
      className={cn(
        cellProps.className,
        "relative p-0 text-center text-sm focus-within:relative focus-within:z-20 [&:has([aria-selected])]:bg-accent first:[&:has([aria-selected])]:rounded-l-md last:[&:has([aria-selected])]:rounded-r-md",
      )}
    >
      <Button
        {...buttonProps}
        type="button"
        variant={"ghost"}
        ref={ref}
        className={cn(
          buttonProps.className,
          "h-9 w-9",
          isToday ? "bg-accent text-accent-foreground" : "",
          isSelected
            ? "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground"
            : "",
          isOutsideVisibleRange ? "text-muted-foreground opacity-50" : "",
          isDisabled ? "text-muted-foreground opacity-50" : "",
        )}
      >
        {formattedDate}
      </Button>
    </td>
  );
}

export { Calendar };
