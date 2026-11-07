import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";

const SECONDS_PER_DAY = 86_400;

/**
 * A time the way a dispatcher reads it on a rolling window: the clock alone when it falls
 * within the next or last twelve hours, the day and clock when it does not, so "Opens
 * 5:00 AM" can never mean tomorrow without saying so.
 */
export function formatWhen(seconds: number, nowSeconds: number): string {
  return Math.abs(seconds - nowSeconds) < SECONDS_PER_DAY / 2
    ? formatUnixTime(seconds)
    : formatUnixDateTimeShort(seconds);
}
