import { useNow } from "@trenova/shared/hooks/use-now";

/**
 * The visitor's local time as a fractional hour (14:30 is 14.5). It moves on the
 * minute, from the app's shared minute clock, so a tab left open drifts with the day.
 */
export function useLocalHour(): number {
  const date = new Date(useNow("minute") * 1000);
  return date.getHours() + date.getMinutes() / 60;
}
