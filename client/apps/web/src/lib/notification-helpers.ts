import { isToday, isYesterday } from "date-fns";

export const NOTIFICATION_DAY_GROUPS = ["Today", "Yesterday", "This week", "Older"] as const;

export type NotificationDayGroup = (typeof NOTIFICATION_DAY_GROUPS)[number];

const WEEK_SECONDS = 7 * 24 * 60 * 60;

/** The heading a moment (Unix seconds) falls under in a list grouped by day. */
export function getNotificationDayGroup(unixSeconds: number): NotificationDayGroup {
  const date = new Date(unixSeconds * 1000);
  if (isToday(date)) return "Today";
  if (isYesterday(date)) return "Yesterday";
  if (Date.now() / 1000 - unixSeconds < WEEK_SECONDS) return "This week";
  return "Older";
}
