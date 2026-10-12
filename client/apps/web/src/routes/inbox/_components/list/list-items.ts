import type { InboundMessageRow, TriageSort } from "@/lib/graphql/inbox";
import { isWaiting } from "@/lib/triage/message-state";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixDateMedium, formatUnixWeekday } from "@trenova/shared/lib/date";
import { groupByDay, type DayGroup, type StartOfDay } from "../day-groups";
import { groupByDue, type DueGroup } from "../due";

export type ListHeading = {
  type: "heading";
  key: string;
  label: string;
  /** "danger" for the over-target group, so the heading says it before the rows do. */
  tone: "default" | "danger";
};

export type ListMessage = { type: "message"; key: string; message: InboundMessageRow };

export type ListItem = ListHeading | ListMessage;

function dueHeading(t: TranslateFn, group: DueGroup): string {
  switch (group) {
    case "over":
      return t("Over target");
    case "hour":
      return t("Due within the hour");
    case "later":
      return t("Later");
  }
}

function dayHeading(t: TranslateFn, group: DayGroup<InboundMessageRow>): string {
  switch (group.bucket) {
    case "today":
      return t("Today");
    case "yesterday":
      return t("Yesterday");
    case "week":
      return formatUnixWeekday(group.day);
    case "earlier":
      return formatUnixDateMedium(group.day);
  }
}

/**
 * A folder as the list draws it: headings between groups and a row per
 * message. Read due first, the groups are over target, due within the hour
 * and later; read newest first, they are the days the mail arrived.
 */
export function buildListItems(
  t: TranslateFn,
  messages: readonly InboundMessageRow[],
  sort: TriageSort,
  now: number,
  startOfDay?: StartOfDay,
): ListItem[] {
  const items: ListItem[] = [];
  const push = (
    key: string,
    label: string,
    tone: ListHeading["tone"],
    rows: readonly InboundMessageRow[],
  ) => {
    items.push({ type: "heading", key: `heading:${key}`, label, tone });
    for (const message of rows) {
      items.push({ type: "message", key: message.id, message });
    }
  };

  if (sort === "Due") {
    const rows = messages.map((message) => ({
      message,
      dueAt: message.dueAt,
      waiting: isWaiting(message),
    }));
    for (const group of groupByDue(rows, now)) {
      push(
        group.group,
        dueHeading(t, group.group),
        group.group === "over" ? "danger" : "default",
        group.items.map((row) => row.message),
      );
    }
    return items;
  }

  for (const group of groupByDay(messages, now, startOfDay)) {
    push(String(group.day), dayHeading(t, group), "default", group.items);
  }

  return items;
}
