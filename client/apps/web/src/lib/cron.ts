import { dateTimeFormatter, formatList } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";

const SUNDAY_EPOCH_MS = Date.UTC(2023, 0, 1);
const DAY_MS = 86_400_000;

export function weekdayName(day: number): string {
  return dateTimeFormatter({ weekday: "long", timeZone: "UTC" }).format(
    new Date(SUNDAY_EPOCH_MS + (day % 7) * DAY_MS),
  );
}

function parseNumber(field: string, min: number, max: number): number | null {
  if (!/^\d+$/.test(field)) return null;
  const value = Number(field);
  return value >= min && value <= max ? value : null;
}

function formatTime(hour: number, minute: number): string {
  const period = hour < 12 ? "AM" : "PM";
  const displayHour = hour % 12 === 0 ? 12 : hour % 12;
  return `${displayHour}:${String(minute).padStart(2, "0")} ${period}`;
}

function ordinal(day: number): string {
  const mod100 = day % 100;
  if (mod100 >= 11 && mod100 <= 13) return `${day}th`;
  switch (day % 10) {
    case 1:
      return `${day}st`;
    case 2:
      return `${day}nd`;
    case 3:
      return `${day}rd`;
    default:
      return `${day}th`;
  }
}

function weekdayList(field: string): string | null {
  const names: string[] = [];
  for (const part of field.split(",")) {
    const day = parseNumber(part, 0, 7);
    if (day === null) return null;
    names.push(weekdayName(day));
  }
  if (names.length === 0) return null;
  return formatList(names);
}

/**
 * Renders a human sentence for the common 5-field cron shapes a schedule UI
 * produces. Returns null for anything it cannot describe faithfully — callers
 * should fall back to showing the raw expression.
 */
export function describeCron(expression: string): string | null {
  const fields = expression.trim().split(/\s+/);
  if (fields.length !== 5) return null;
  const [minuteField, hourField, domField, monthField, dowField] = fields;

  const minute = parseNumber(minuteField, 0, 59);
  const hour = parseNumber(hourField, 0, 23);
  if (minute === null || hour === null || monthField !== "*") return null;

  const time = formatTime(hour, minute);

  if (domField === "*" && dowField === "*") {
    return translate("Daily at {0}", time);
  }

  if (domField === "*" && dowField === "1-5") {
    return translate("Weekdays at {0}", time);
  }

  if (domField === "*") {
    const days = weekdayList(dowField);
    return days ? translate("Weekly on {0} at {1}", days, time) : null;
  }

  if (dowField === "*") {
    const day = parseNumber(domField, 1, 31);
    return day !== null ? translate("Monthly on the {0} at {1}", ordinal(day), time) : null;
  }

  return null;
}

export function ordinalDay(day: number): string {
  return ordinal(day);
}

export function formatTimeOfDay(hour: number, minute: number): string {
  return formatTime(hour, minute);
}

export type CronFrequency = "daily" | "weekly" | "monthly";

export type CronParts = {
  frequency: CronFrequency;
  hour: number;
  minute: number;
  weekdays: number[];
  dayOfMonth: number;
};

export const DEFAULT_CRON_PARTS: CronParts = {
  frequency: "weekly",
  hour: 8,
  minute: 0,
  weekdays: [1],
  dayOfMonth: 1,
};

function parseWeekdays(field: string): number[] | null {
  if (field === "1-5") return [1, 2, 3, 4, 5];
  const days = new Set<number>();
  for (const part of field.split(",")) {
    const day = parseNumber(part, 0, 7);
    if (day === null) return null;
    days.add(day % 7);
  }
  return days.size > 0 ? [...days].sort((a, b) => a - b) : null;
}

/**
 * Inverse of {@link buildCron}. Parses the friendly cron shapes the cadence
 * builder produces into structured parts. Returns null for any expression the
 * builder cannot round-trip — callers fall back to the raw (advanced) editor.
 */
export function parseCron(expression: string): CronParts | null {
  const fields = expression.trim().split(/\s+/);
  if (fields.length !== 5) return null;
  const [minuteField, hourField, domField, monthField, dowField] = fields;

  const minute = parseNumber(minuteField, 0, 59);
  const hour = parseNumber(hourField, 0, 23);
  if (minute === null || hour === null || monthField !== "*") return null;

  const base = { ...DEFAULT_CRON_PARTS, hour, minute };

  if (domField === "*" && dowField === "*") {
    return { ...base, frequency: "daily" };
  }

  if (domField === "*") {
    const weekdays = parseWeekdays(dowField);
    return weekdays ? { ...base, frequency: "weekly", weekdays } : null;
  }

  if (dowField === "*") {
    const day = parseNumber(domField, 1, 31);
    return day !== null ? { ...base, frequency: "monthly", dayOfMonth: day } : null;
  }

  return null;
}

export function buildCron(parts: CronParts): string {
  const { frequency, hour, minute, dayOfMonth } = parts;

  switch (frequency) {
    case "daily":
      return `${minute} ${hour} * * *`;
    case "monthly":
      return `${minute} ${hour} ${dayOfMonth} * *`;
    case "weekly": {
      const days = [...new Set(parts.weekdays)].sort((a, b) => a - b);
      const normalized = days.length > 0 ? days : [1];
      const isWeekdays = normalized.length === 5 && normalized.join(",") === "1,2,3,4,5";
      return `${minute} ${hour} * * ${isWeekdays ? "1-5" : normalized.join(",")}`;
    }
  }
}
