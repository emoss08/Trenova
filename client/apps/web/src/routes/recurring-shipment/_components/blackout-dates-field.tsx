import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { fromISODateString, toISODateString } from "@trenova/shared/lib/date";
import { holidayNamesByDate, usFederalHolidays } from "@trenova/shared/lib/holidays";
import { cn } from "@trenova/shared/lib/utils";
import { FieldWrapper } from "@/components/fields/field-components";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { format } from "date-fns";
import { CalendarPlusIcon, ChevronDownIcon, SparklesIcon, XIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useController, useFormContext } from "react-hook-form";

/** Mirrors `recurringshipment.MaxBlackoutDates` on the server. */
const MAX_BLACKOUT_DATES = 100;

const HOLIDAY_YEARS_AHEAD = 2;
const CALENDAR_YEARS_BEHIND = 1;
const CALENDAR_YEARS_AHEAD = 5;

type BlackoutYearGroup = {
  year: number;
  dates: string[];
};

function groupByYear(dates: readonly string[]): BlackoutYearGroup[] {
  const groups: BlackoutYearGroup[] = [];
  for (const date of dates) {
    const year = Number(date.slice(0, 4));
    const current = groups[groups.length - 1];
    if (current?.year === year) {
      current.dates.push(date);
      continue;
    }
    groups.push({ year, dates: [date] });
  }
  return groups;
}

type BlackoutDateRowProps = {
  date: string;
  holidayName?: string;
  isPast: boolean;
  onRemove: (date: string) => void;
};

function BlackoutDateRow({ date, holidayName, isPast, onRemove }: BlackoutDateRowProps) {
  const parsed = fromISODateString(date);
  const label = parsed ? format(parsed, "MMM d") : date;

  return (
    <li className="group/row hover:bg-muted flex items-center gap-2 rounded-md px-2 py-1 transition-colors">
      <span className="text-2xs text-muted-foreground w-8 shrink-0 font-medium tracking-wide uppercase">
        {parsed ? format(parsed, "EEE") : "—"}
      </span>
      <span className={cn("text-sm tabular-nums", isPast && "text-muted-foreground")}>{label}</span>
      {holidayName && (
        <Badge
          variant="outline"
          className="text-2xs h-4 border-none bg-blue-600/10 px-1.5 font-normal text-blue-600 dark:text-blue-400"
        >
          {holidayName}
        </Badge>
      )}
      {isPast && <span className="text-2xs text-muted-foreground">Past</span>}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={`Remove ${label}`}
        onClick={() => onRemove(date)}
        className="text-muted-foreground hover:text-foreground ml-auto size-5 opacity-0 transition-opacity group-focus-within/row:opacity-100 group-hover/row:opacity-100 focus-visible:opacity-100"
      >
        <XIcon className="size-3.5" />
      </Button>
    </li>
  );
}

/**
 * The blackout calendar.
 *
 * Blackout dates are stored as `YYYY-MM-DD` calendar days rather than instants,
 * so everything here round-trips through the local-date helpers. In practice
 * the set is "the days our facilities are closed", which is why holidays are a
 * one-click preset instead of eleven trips through a date picker.
 */
export function BlackoutDatesField() {
  const { control } = useFormContext<RecurringShipment>();
  const { field, fieldState } = useController({ control, name: "blackoutDates" });

  const dates = useMemo(() => field.value ?? [], [field.value]);
  const todayISO = useMemo(() => toISODateString(new Date()), []);
  const currentYear = useMemo(() => new Date().getFullYear(), []);

  const commit = useCallback(
    (next: readonly string[]) => {
      field.onChange([...new Set(next)].sort().slice(0, MAX_BLACKOUT_DATES));
    },
    [field],
  );

  const selectedDays = useMemo(
    () => dates.map(fromISODateString).filter((date): date is Date => date !== null),
    [dates],
  );

  const holidayNames = useMemo(() => {
    const years = new Set<number>(dates.map((date) => Number(date.slice(0, 4))));
    for (let offset = 0; offset <= HOLIDAY_YEARS_AHEAD; offset++) {
      years.add(currentYear + offset);
    }
    return holidayNamesByDate([...years]);
  }, [currentYear, dates]);

  const holidayYears = useMemo(
    () =>
      Array.from({ length: HOLIDAY_YEARS_AHEAD + 1 }, (_, offset) => {
        const year = currentYear + offset;
        const missing = usFederalHolidays(year).filter(
          (holiday) => !dates.includes(holiday.date),
        ).length;
        return { year, missing };
      }),
    [currentYear, dates],
  );

  const groups = useMemo(() => groupByYear(dates), [dates]);
  const pastDates = useMemo(() => dates.filter((date) => date < todayISO), [dates, todayISO]);

  const handleSelect = useCallback(
    (days: Date[] | undefined) => {
      commit((days ?? []).map(toISODateString));
    },
    [commit],
  );

  const addHolidays = useCallback(
    (year: number) => {
      commit([...dates, ...usFederalHolidays(year).map((holiday) => holiday.date)]);
    },
    [commit, dates],
  );

  const removeDate = useCallback(
    (value: string) => {
      commit(dates.filter((date) => date !== value));
    },
    [commit, dates],
  );

  const atCapacity = dates.length >= MAX_BLACKOUT_DATES;

  return (
    <FieldWrapper
      label="Blackout Dates"
      description="Days your facilities are closed. An occurrence landing on one of these follows the exception policy instead of generating a shipment."
      error={fieldState.error?.message}
    >
      <div className="border-input bg-muted/30 overflow-hidden rounded-lg border">
        <div className="border-input bg-muted/50 flex flex-wrap items-center gap-1.5 border-b p-1.5">
          <Popover>
            <PopoverTrigger
              render={
                <Button type="button" variant="outline" size="sm" className="h-7">
                  <CalendarPlusIcon className="size-3.5" />
                  Select days
                </Button>
              }
            />
            <PopoverContent align="start" className="w-auto p-0">
              <Calendar
                mode="multiple"
                selected={selectedDays}
                onSelect={handleSelect}
                max={MAX_BLACKOUT_DATES}
                className="w-full"
                defaultMonth={selectedDays[0]}
                captionLayout="dropdown"
                startMonth={new Date(currentYear - CALENDAR_YEARS_BEHIND, 0)}
                endMonth={new Date(currentYear + CALENDAR_YEARS_AHEAD, 11)}
                showOutsideDays={false}
              />
              <p className="border-border text-2xs text-muted-foreground border-t px-3 py-2">
                Click a day to block it, click it again to unblock.
              </p>
            </PopoverContent>
          </Popover>

          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7"
                  disabled={atCapacity}
                >
                  <SparklesIcon className="size-3.5" />
                  Holidays
                  <ChevronDownIcon className="text-muted-foreground size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent align="start">
              <DropdownMenuGroup>
                <DropdownMenuLabel>Add US federal holidays</DropdownMenuLabel>
                {holidayYears.map(({ year, missing }) => (
                  <DropdownMenuItem
                    key={year}
                    title={String(year)}
                    description={missing > 0 ? `Adds ${missing} days` : "Already blocked"}
                    disabled={missing === 0}
                    onClick={() => addHolidays(year)}
                  />
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <span className="text-2xs text-muted-foreground ml-auto pr-1 tabular-nums">
            {atCapacity ? `Limit ${MAX_BLACKOUT_DATES} reached` : `${dates.length} blocked`}
          </span>
          {dates.length > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="text-muted-foreground hover:text-foreground h-7"
              onClick={() => commit([])}
            >
              Clear
            </Button>
          )}
        </div>

        {dates.length === 0 ? (
          <div className="flex flex-col items-center gap-0.5 px-3 py-6 text-center">
            <p className="text-sm">No days blocked</p>
            <p className="text-2xs text-muted-foreground">
              Every scheduled occurrence generates a shipment.
            </p>
          </div>
        ) : (
          <ScrollArea className="max-h-56" maskVariant="muted">
            {pastDates.length > 0 && (
              <div className="border-border flex items-center gap-2 border-b px-2.5 py-1.5">
                <span className="text-2xs text-muted-foreground">
                  {pastDates.length} {pastDates.length === 1 ? "day has" : "days have"} already
                  passed and no longer affect the schedule.
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="text-2xs ml-auto h-6"
                  onClick={() => commit(dates.filter((date) => date >= todayISO))}
                >
                  Remove
                </Button>
              </div>
            )}
            <div className="p-1.5">
              {groups.map((group) => (
                <div key={group.year} className="not-first:mt-2">
                  <div className="text-2xs text-muted-foreground px-2 pb-1 font-medium tracking-wide">
                    {group.year}
                  </div>
                  <ul className="flex flex-col">
                    {group.dates.map((date) => (
                      <BlackoutDateRow
                        key={date}
                        date={date}
                        holidayName={holidayNames.get(date)}
                        isPast={date < todayISO}
                        onRemove={removeDate}
                      />
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </div>
    </FieldWrapper>
  );
}
