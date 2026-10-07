const WEEKDAY_NAMES = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];

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
    names.push(WEEKDAY_NAMES[day % 7]);
  }
  if (names.length === 0) return null;
  if (names.length === 1) return names[0];
  if (names.length === 2) return `${names[0]} and ${names[1]}`;
  return `${names.slice(0, -1).join(", ")}, and ${names[names.length - 1]}`;
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
    return `Daily at ${time}`;
  }

  if (domField === "*" && dowField === "1-5") {
    return `Weekdays at ${time}`;
  }

  if (domField === "*") {
    const days = weekdayList(dowField);
    return days ? `Weekly on ${days} at ${time}` : null;
  }

  if (dowField === "*") {
    const day = parseNumber(domField, 1, 31);
    return day !== null ? `Monthly on the ${ordinal(day)} at ${time}` : null;
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

/** A five-field expression read the way the server's parser reads it. */
export type CronSchedule = {
  minutes: ReadonlySet<number>;
  hours: ReadonlySet<number>;
  daysOfMonth: ReadonlySet<number>;
  months: ReadonlySet<number>;
  daysOfWeek: ReadonlySet<number>;
  /** Day of month was "*": then only the day of week decides the day, and the other way round. */
  anyDayOfMonth: boolean;
  anyDayOfWeek: boolean;
};

const MONTH_NAMES = ["JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"];
const DAY_NAMES = ["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"];

function fieldValue(raw: string, min: number, names?: readonly string[]): number | null {
  if (/^\d+$/.test(raw)) {
    return Number(raw);
  }
  const at = names?.indexOf(raw.toUpperCase()) ?? -1;
  return at < 0 ? null : at + min;
}

function parseField(
  field: string,
  min: number,
  max: number,
  names?: readonly string[],
): Set<number> | null {
  const values = new Set<number>();
  for (const part of field.split(",")) {
    const [range, stepText] = part.split("/");
    const step = stepText === undefined ? 1 : Number(stepText);
    if (!Number.isInteger(step) || step < 1 || range === undefined || range === "") {
      return null;
    }
    let low: number | null;
    let high: number | null;
    if (range === "*" || range === "?") {
      low = min;
      high = max;
    } else if (range.includes("-")) {
      const [from, to] = range.split("-");
      low = fieldValue(from ?? "", min, names);
      high = fieldValue(to ?? "", min, names);
    } else {
      low = fieldValue(range, min, names);
      high = stepText === undefined ? low : max;
    }
    if (low === null || high === null || low < min || high > max || low > high) {
      return null;
    }
    for (let value = low; value <= high; value += step) {
      values.add(value);
    }
  }
  return values.size ? values : null;
}

/** Reads a standard five-field expression; null for anything the server would refuse. */
export function parseCronSchedule(expression: string): CronSchedule | null {
  const fields = expression.trim().split(/\s+/);
  if (fields.length !== 5) return null;
  const [minuteField, hourField, domField, monthField, dowField] = fields as [
    string,
    string,
    string,
    string,
    string,
  ];
  const minutes = parseField(minuteField, 0, 59);
  const hours = parseField(hourField, 0, 23);
  const daysOfMonth = parseField(domField, 1, 31);
  const months = parseField(monthField, 1, 12, MONTH_NAMES);
  const rawDays = parseField(dowField, 0, 7, DAY_NAMES);
  if (!minutes || !hours || !daysOfMonth || !months || !rawDays) return null;
  const daysOfWeek = new Set([...rawDays].map((day) => day % 7));
  return {
    minutes,
    hours,
    daysOfMonth,
    months,
    daysOfWeek,
    anyDayOfMonth: domField === "*" || domField === "?",
    anyDayOfWeek: dowField === "*" || dowField === "?",
  };
}

export type WallClock = {
  year: number;
  month: number;
  day: number;
  weekday: number;
  hour: number;
  minute: number;
};

/** Whether a wall-clock minute is one the schedule fires on. */
export function cronFiresAt(schedule: CronSchedule, at: WallClock): boolean {
  if (
    !schedule.minutes.has(at.minute) ||
    !schedule.hours.has(at.hour) ||
    !schedule.months.has(at.month)
  ) {
    return false;
  }
  const dom = schedule.daysOfMonth.has(at.day);
  const dow = schedule.daysOfWeek.has(at.weekday);
  if (schedule.anyDayOfMonth || schedule.anyDayOfWeek) {
    return dom && dow;
  }
  return dom || dow;
}

const WEEKDAY_INDEX: Record<string, number> = {
  Sun: 0,
  Mon: 1,
  Tue: 2,
  Wed: 3,
  Thu: 4,
  Fri: 5,
  Sat: 6,
};

const wallClockFormatters = new Map<string, Intl.DateTimeFormat>();

/** The wall clock in a time zone at a Unix second. */
export function wallClockAt(unix: number, timeZone: string): WallClock {
  let formatter = wallClockFormatters.get(timeZone);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat("en-US", {
      timeZone,
      hourCycle: "h23",
      year: "numeric",
      month: "numeric",
      day: "numeric",
      weekday: "short",
      hour: "numeric",
      minute: "numeric",
    });
    wallClockFormatters.set(timeZone, formatter);
  }
  const parts: Record<string, string> = {};
  for (const part of formatter.formatToParts(new Date(unix * 1000))) {
    parts[part.type] = part.value;
  }
  return {
    year: Number(parts.year),
    month: Number(parts.month),
    day: Number(parts.day),
    weekday: WEEKDAY_INDEX[parts.weekday ?? "Sun"] ?? 0,
    hour: Number(parts.hour) % 24,
    minute: Number(parts.minute),
  };
}

/** Every minute in [from, until) the schedule fires on in a time zone, as Unix seconds. */
export function cronRunsBetween(
  schedule: CronSchedule,
  timeZone: string,
  from: number,
  until: number,
): number[] {
  const runs: number[] = [];
  for (let at = from - (from % 60); at < until; at += 60) {
    if (at >= from && cronFiresAt(schedule, wallClockAt(at, timeZone))) {
      runs.push(at);
    }
  }
  return runs;
}
