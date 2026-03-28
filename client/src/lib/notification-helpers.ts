import type { BadgeVariant } from "@/components/ui/badge";
import { formatDistanceToNowStrict } from "date-fns";
import {
  CircleAlertIcon,
  CircleCheckIcon,
  InfoIcon,
  TriangleAlertIcon,
} from "lucide-react";
import { createElement } from "react";

export function formatTimestamp(unixSeconds: number): string {
  return formatDistanceToNowStrict(new Date(unixSeconds * 1000), {
    addSuffix: true,
  });
}

export const SOURCE_LABELS: Record<string, string> = {
  table_change_alert: "Table Change",
};

export const PRIORITY_CONFIG: Record<
  string,
  { icon: React.ReactNode; badge: BadgeVariant; dot: string }
> = {
  critical: {
    icon: createElement(CircleAlertIcon, { className: "size-4 text-red-500" }),
    badge: "inactive",
    dot: "bg-red-500",
  },
  high: {
    icon: createElement(TriangleAlertIcon, { className: "size-4 text-orange-500" }),
    badge: "orange",
    dot: "bg-orange-500",
  },
  medium: {
    icon: createElement(InfoIcon, { className: "size-4 text-blue-500" }),
    badge: "info",
    dot: "bg-blue-500",
  },
  low: {
    icon: createElement(CircleCheckIcon, { className: "size-4 text-muted-foreground" }),
    badge: "outline",
    dot: "bg-muted-foreground",
  },
};

export function getPriorityConfig(priority: string) {
  return PRIORITY_CONFIG[priority] ?? PRIORITY_CONFIG.medium;
}
