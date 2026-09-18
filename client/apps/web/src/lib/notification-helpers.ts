import type { BadgeVariant } from "@trenova/shared/components/ui/badge";
import { formatDistanceToNowStrict, isToday, isYesterday } from "date-fns";
import { CircleAlertIcon, CircleCheckIcon, InfoIcon, TriangleAlertIcon } from "lucide-react";
import { createElement } from "react";

export function formatTimestamp(unixSeconds: number): string {
  return formatDistanceToNowStrict(new Date(unixSeconds * 1000), {
    addSuffix: true,
  });
}

export const NOTIFICATION_DAY_GROUPS = ["Today", "Yesterday", "This week", "Older"] as const;

export type NotificationDayGroup = (typeof NOTIFICATION_DAY_GROUPS)[number];

const WEEK_SECONDS = 7 * 24 * 60 * 60;

export function getNotificationDayGroup(unixSeconds: number): NotificationDayGroup {
  const date = new Date(unixSeconds * 1000);
  if (isToday(date)) return "Today";
  if (isYesterday(date)) return "Yesterday";
  if (Date.now() / 1000 - unixSeconds < WEEK_SECONDS) return "This week";
  return "Older";
}

export const SOURCE_LABELS: Record<string, string> = {
  table_change_alert: "Table Change",
};

export const PRIORITY_CONFIG: Record<
  string,
  { icon: React.ReactNode; badge: BadgeVariant; dot: string }
> = {
  critical: {
    icon: createElement(CircleAlertIcon, { className: "size-4 text-danger-foreground" }),
    badge: "danger",
    dot: "bg-danger",
  },
  high: {
    icon: createElement(TriangleAlertIcon, { className: "size-4 text-warning-foreground" }),
    badge: "warning",
    dot: "bg-warning",
  },
  medium: {
    icon: createElement(InfoIcon, { className: "size-4 text-info-foreground" }),
    badge: "info",
    dot: "bg-info",
  },
  low: {
    icon: createElement(CircleCheckIcon, { className: "size-4 text-muted-foreground" }),
    badge: "neutral",
    dot: "bg-muted-foreground",
  },
};

export function getPriorityConfig(priority: string) {
  return PRIORITY_CONFIG[priority] ?? PRIORITY_CONFIG.medium;
}
