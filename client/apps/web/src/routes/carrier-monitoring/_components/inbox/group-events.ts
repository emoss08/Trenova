import { SECONDS_PER_DAY } from "@/lib/carrier-intelligence";
import type { CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import { formatUnixDateMedium, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { InboxGrouping } from "./inbox-filters";

export type InboxEventGroup = {
  key: string;
  label: string;
  events: CarrierIntelEvent[];
};

export type GroupInboxEventsParams = {
  events: readonly CarrierIntelEvent[];
  grouping: InboxGrouping;
  now: number;
  labels: { today: string; yesterday: string; usdot: (dotNumber: string) => string };
};

function dayKey(timestamp: number): string {
  return formatUnixInUserTimezone(timestamp, { year: "numeric", month: "2-digit", day: "2-digit" });
}

export function groupInboxEvents({
  events,
  grouping,
  now,
  labels,
}: GroupInboxEventsParams): InboxEventGroup[] {
  const groups = new Map<string, InboxEventGroup>();

  if (grouping === "carrier") {
    for (const event of events) {
      const key = event.carrierId ?? `dot:${event.dotNumber}`;
      let group = groups.get(key);
      if (!group) {
        group = { key, label: event.subjectName || labels.usdot(event.dotNumber), events: [] };
        groups.set(key, group);
      }
      group.events.push(event);
    }
    return [...groups.values()];
  }

  const today = dayKey(now);
  const yesterday = dayKey(now - SECONDS_PER_DAY);
  for (const event of events) {
    const key = dayKey(event.detectedAt);
    let group = groups.get(key);
    if (!group) {
      const label =
        key === today
          ? labels.today
          : key === yesterday
            ? labels.yesterday
            : formatUnixDateMedium(event.detectedAt);
      group = { key, label, events: [] };
      groups.set(key, group);
    }
    group.events.push(event);
  }
  return [...groups.values()];
}
