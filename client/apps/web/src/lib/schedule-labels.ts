import type { ConversationSchedule } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatDateInUserTimezone } from "@trenova/shared/lib/date";

const DAY_SECONDS = 24 * 60 * 60;

/** The calendar day an instant falls on in a timezone, as yyyy-mm-dd. */
function dayKey(unixSeconds: number, timezone: string): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(unixSeconds * 1000));
}

/** A time as the reader's language and clock write it, without the narrow space newer ICU puts before AM/PM. */
function clockTime(date: Date, timezone: string): string {
  return formatDateInUserTimezone(date, { hour: "numeric", minute: "2-digit", timezone }).replace(
    /\u202f/g,
    " ",
  );
}

/**
 * When a schedule runs next, or last ran, the way its card says it: "Today ·
 * 7:30 AM", "Tomorrow · 8:00 AM", or "Mon, Oct 5 · 7:30 AM". Read on the
 * reader's clock and written in their language.
 */
export function nextRunLabel(
  nextRunAt: number,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  const at = new Date(nextRunAt * 1000);
  const time = clockTime(at, timezone);

  const day = dayKey(nextRunAt, timezone);
  if (day === dayKey(now, timezone)) {
    return t("Today · {0}", time);
  }
  if (day === dayKey(now + DAY_SECONDS, timezone)) {
    return t("Tomorrow · {0}", time);
  }

  const date = formatDateInUserTimezone(at, {
    timezone,
    weekday: "short",
    month: "short",
    day: "numeric",
  });

  return `${date} · ${time}`;
}

const CRON_NUMBER = /^\d{1,2}$/;

/**
 * When a schedule runs, in the reader's language: "Every weekday · 7:30 AM".
 * It is written from the schedule's cron, the time as the cron names it on
 * the schedule's own clock; the English label the server stored is the
 * fallback for a cron this does not describe.
 */
export function cadenceLabel(
  schedule: Pick<ConversationSchedule, "cadence" | "cronExpression">,
  t: TranslateFn,
): string {
  const fields = schedule.cronExpression.trim().split(/\s+/);
  if (fields.length !== 5) {
    return schedule.cadence;
  }
  const [minuteField, hourField, dayOfMonth, month, dayOfWeek] = fields;
  if (!CRON_NUMBER.test(minuteField) || !CRON_NUMBER.test(hourField)) {
    return schedule.cadence;
  }
  const hour = Number(hourField);
  const minute = Number(minuteField);
  if (hour > 23 || minute > 59 || dayOfMonth !== "*" || month !== "*") {
    return schedule.cadence;
  }
  const time = clockTime(new Date(Date.UTC(2000, 0, 1, hour, minute)), "UTC");

  switch (dayOfWeek) {
    case "*":
      return t("Every day · {0}", time);
    case "1-5":
      return t("Every weekday · {0}", time);
    case "0":
    case "7":
      return t("Every Sunday · {0}", time);
    case "1":
      return t("Every Monday · {0}", time);
    case "2":
      return t("Every Tuesday · {0}", time);
    case "3":
      return t("Every Wednesday · {0}", time);
    case "4":
      return t("Every Thursday · {0}", time);
    case "5":
      return t("Every Friday · {0}", time);
    case "6":
      return t("Every Saturday · {0}", time);
    default:
      return schedule.cadence;
  }
}

/** The line under a schedule's request: "{when} · next {next}", or "paused". */
export function scheduleLine(
  schedule: Pick<ConversationSchedule, "cadence" | "cronExpression" | "enabled" | "nextRunAt">,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  const cadence = cadenceLabel(schedule, t);
  if (!schedule.enabled) {
    return t("{0} · paused", cadence);
  }
  if (!schedule.nextRunAt) {
    return cadence;
  }

  return t("{0} · next {1}", cadence, nextRunLabel(schedule.nextRunAt, now, timezone, t));
}
