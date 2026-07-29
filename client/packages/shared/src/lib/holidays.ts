import { toISODateString } from "@trenova/shared/lib/date";

export type Holiday = {
  /** Calendar day as `YYYY-MM-DD`. */
  date: string;
  name: string;
};

type FixedRule = { name: string; month: number; day: number };
type NthWeekdayRule = { name: string; month: number; weekday: number; nth: number };

/** Months are 1-indexed here to read like a calendar; weekdays are 0 = Sunday. */
const FIXED_RULES: readonly FixedRule[] = [
  { name: "New Year's Day", month: 1, day: 1 },
  { name: "Juneteenth", month: 6, day: 19 },
  { name: "Independence Day", month: 7, day: 4 },
  { name: "Veterans Day", month: 11, day: 11 },
  { name: "Christmas Day", month: 12, day: 25 },
];

/** A negative `nth` counts back from the end of the month (-1 = last). */
const NTH_WEEKDAY_RULES: readonly NthWeekdayRule[] = [
  { name: "Martin Luther King Jr. Day", month: 1, weekday: 1, nth: 3 },
  { name: "Presidents' Day", month: 2, weekday: 1, nth: 3 },
  { name: "Memorial Day", month: 5, weekday: 1, nth: -1 },
  { name: "Labor Day", month: 9, weekday: 1, nth: 1 },
  { name: "Columbus Day", month: 10, weekday: 1, nth: 2 },
  { name: "Thanksgiving Day", month: 11, weekday: 4, nth: 4 },
];

function nthWeekdayOfMonth(year: number, month: number, weekday: number, nth: number): Date {
  if (nth < 0) {
    const lastDay = new Date(year, month, 0);
    const shift = (lastDay.getDay() - weekday + 7) % 7;
    return new Date(year, month - 1, lastDay.getDate() - shift - (Math.abs(nth) - 1) * 7);
  }
  const firstDay = new Date(year, month - 1, 1);
  const shift = (weekday - firstDay.getDay() + 7) % 7;
  return new Date(year, month - 1, 1 + shift + (nth - 1) * 7);
}

/**
 * The eleven US federal holidays for a calendar year, in date order.
 *
 * These are the observance dates carriers actually shut down on, so they double
 * as the default blackout set for a recurring lane. The federal "observed"
 * shift for holidays landing on a weekend is deliberately not applied: freight
 * follows the facility's closure, not the government's payroll calendar, and a
 * blackout on a day the lane never runs is inert anyway.
 */
export function usFederalHolidays(year: number): Holiday[] {
  const holidays: Holiday[] = [
    ...FIXED_RULES.map((rule) => ({
      name: rule.name,
      date: toISODateString(new Date(year, rule.month - 1, rule.day)),
    })),
    ...NTH_WEEKDAY_RULES.map((rule) => ({
      name: rule.name,
      date: toISODateString(nthWeekdayOfMonth(year, rule.month, rule.weekday, rule.nth)),
    })),
  ];

  return holidays.sort((a, b) => a.date.localeCompare(b.date));
}

/**
 * Maps `YYYY-MM-DD` to a holiday name across the given years, so an arbitrary
 * set of dates can be labelled with what the day actually is.
 */
export function holidayNamesByDate(years: readonly number[]): Map<string, string> {
  const names = new Map<string, string>();
  for (const year of years) {
    for (const holiday of usFederalHolidays(year)) {
      names.set(holiday.date, holiday.name);
    }
  }
  return names;
}
