import { useNow } from "@trenova/shared/hooks/use-now";
import { useSearchParams } from "react-router";

/** The query parameter that pins the stage's hour in development (`?hour=17:30`). */
export const HOUR_OVERRIDE_PARAM = "hour";

const CLOCK_PATTERN = /^(\d{1,2}):([0-5]\d)$/;
const DECIMAL_PATTERN = /^\d{1,2}(?:\.\d+)?$/;

/**
 * Reads a pinned hour as `H`, `H.5` or `HH:MM`, from 0 up to but not including 24.
 * Null for anything else, so a malformed value falls back to the clock.
 */
export function parseHourOverride(value: string | null): number | null {
  if (!value) {
    return null;
  }
  const clock = CLOCK_PATTERN.exec(value);
  const hour = clock
    ? Number(clock[1]) + Number(clock[2]) / 60
    : DECIMAL_PATTERN.test(value)
      ? Number(value)
      : Number.NaN;
  return hour >= 0 && hour < 24 ? hour : null;
}

/**
 * The visitor's local time as a fractional hour (14:30 is 14.5). It moves on the
 * minute, from the app's shared minute clock, so a tab left open drifts with the day.
 * In development `?hour=` pins it, so each palette can be checked without waiting
 * for the day to reach it; production builds ignore the parameter.
 */
export function useLocalHour(): number {
  const [searchParams] = useSearchParams();
  const date = new Date(useNow("minute") * 1000);
  const override = import.meta.env.DEV
    ? parseHourOverride(searchParams.get(HOUR_OVERRIDE_PARAM))
    : null;
  return override ?? date.getHours() + date.getMinutes() / 60;
}
