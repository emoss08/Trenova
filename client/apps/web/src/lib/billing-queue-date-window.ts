import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, fromUserWallClock, userWallClockNow } from "@trenova/shared/lib/date";
import type { BillingQueueSearchDates } from "@trenova/shared/types/billing-queue";

/**
 * A window of days a billing queue filter keeps: the last N days up to today, or a
 * range of days chosen on a calendar. Each bound is the Unix start of a day in the
 * person's timezone; the server keeps every moment of the last day.
 */
export type DateWindow =
  | { kind: "last"; days: number }
  | { kind: "range"; from: number; to: number };

/** The rolling windows the filter offers before a custom range. */
export const LAST_DAYS = [7, 30, 90] as const;

/** A window as it sits in the URL and in a saved preset: `last:7` or `range:FROM:TO`. */
export function serializeDateWindow(span: DateWindow): string {
  return span.kind === "last" ? `last:${span.days}` : `range:${span.from}:${span.to}`;
}

/** A window read back from the URL or a preset; anything else is no window. */
export function parseDateWindow(value: string | null | undefined): DateWindow | null {
  if (!value) return null;
  const [kind, first, second] = value.split(":");
  if (kind === "last") {
    const days = Number(first);
    return Number.isInteger(days) && days > 0 ? { kind: "last", days } : null;
  }
  if (kind === "range") {
    const from = Number(first);
    const to = Number(second);
    return Number.isInteger(from) && Number.isInteger(to) && from <= to
      ? { kind: "range", from, to }
      : null;
  }

  return null;
}

/** The list filter that keeps rows whose `field` falls in the window. */
export function dateWindowFilter(
  field: string,
  span: DateWindow,
): { field: string; operator: string; value: unknown } {
  return span.kind === "last"
    ? { field, operator: "lastndays", value: span.days }
    : { field, operator: "daterange", value: { from: span.from, to: span.to } };
}

/** The dates the billing queue filters by. */
export type QueueDateField = "delivered" | "pickedUp" | "queued";

/** A window said as one phrase for its field, for the chip that clears it. */
export function dateWindowLabel(field: QueueDateField, span: DateWindow, t: TranslateFn): string {
  if (span.kind === "last") {
    switch (field) {
      case "delivered":
        return t(
          "{0, plural, one {Delivered in the last day} other {Delivered in the last # days}}",
          span.days,
        );
      case "pickedUp":
        return t(
          "{0, plural, one {Picked up in the last day} other {Picked up in the last # days}}",
          span.days,
        );
      case "queued":
        return t(
          "{0, plural, one {Queued in the last day} other {Queued in the last # days}}",
          span.days,
        );
    }
  }
  const from = formatUnixMonthDay(span.from);
  const to = formatUnixMonthDay(span.to);
  switch (field) {
    case "delivered":
      return span.from === span.to ? t("Delivered {0}", from) : t("Delivered {0} – {1}", from, to);
    case "pickedUp":
      return span.from === span.to ? t("Picked up {0}", from) : t("Picked up {0} – {1}", from, to);
    case "queued":
      return span.from === span.to ? t("Queued {0}", from) : t("Queued {0} – {1}", from, to);
  }
}

const DAY = /^(\d{4})-(\d{2})-(\d{2})$/u;

/** A calendar day (YYYY-MM-DD) as the Unix start of that day in the person's timezone. */
export function dayStart(day: string): number | null {
  const match = DAY.exec(day);
  if (!match) return null;
  const start = fromUserWallClock(
    new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3])),
  );

  return start ?? null;
}

/** Today in the person's calendar, as YYYY-MM-DD, for Nova to count "last week" from. */
export function todayInUserCalendar(): string {
  const now = userWallClockNow();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");

  return `${now.getFullYear()}-${month}-${day}`;
}

/** The window a queue search named, in the person's timezone; null when it named none. */
export function dateWindowFromSearch(
  dates: BillingQueueSearchDates | null | undefined,
): DateWindow | null {
  if (!dates) return null;
  if (dates.lastDays) return { kind: "last", days: dates.lastDays };
  const from = dates.from ? dayStart(dates.from) : null;
  const to = dates.to ? dayStart(dates.to) : null;

  return from != null && to != null && from <= to ? { kind: "range", from, to } : null;
}
