import { formatUnixTime, formatUnixWeekday } from "@trenova/shared/lib/date";
import type { TimelineScale } from "./timeline-scale";

/**
 * A moment as the timeline reads it: the clock alone inside a one-day range,
 * with the weekday in front anywhere else, so a time always says which day.
 */
export function formatTimelineTime(unix: number, scale: TimelineScale): string {
  if (scale.range === "day" && unix >= scale.start && unix < scale.end) return formatUnixTime(unix);
  return [formatUnixWeekday(unix), formatUnixTime(unix)].join(" ");
}

/** A rough caption width at the bar's text size, to decide whether it fits. */
export function captionWidth(text: string): number {
  return text.length * 6.2 + 14;
}
