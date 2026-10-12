import { dateTimeFormatter } from "@trenova/shared/i18n/format";
import type { TimeFormatType } from "@trenova/shared/types/user";
import { addDays, addHours, endOfDay, nextMonday, set } from "date-fns";
import { fromZonedTime, toZonedTime } from "date-fns-tz";

export const PAUSE_CHOICES = ["off", "1h", "tomorrow", "monday"] as const;
export type PauseChoice = (typeof PAUSE_CHOICES)[number] | "current";

export const FOCUS_CHOICES = ["1h", "2h", "eod"] as const;
export type FocusChoice = (typeof FOCUS_CHOICES)[number];

export const WEEKDAYS = ["0", "1", "2", "3", "4", "5", "6"] as const;

const MORNING_HOUR = 8;
const HOUR_SECONDS = 3600;

function toSeconds(date: Date): number {
  return Math.floor(date.getTime() / 1000);
}

/** The wall clock in a zone, as a Date whose local fields read that zone's time. */
function wallClock(now: Date, timezone: string): Date {
  return toZonedTime(now, timezone);
}

/** Unix seconds for a wall-clock time in a zone. */
function fromWallClock(wall: Date, timezone: string): number {
  return toSeconds(fromZonedTime(wall, timezone));
}

/** When a pause chosen now ends: an hour from now, 8:00 tomorrow, or 8:00 next Monday. */
export function pauseUntil(choice: PauseChoice, now: Date, timezone: string): number | null {
  const wall = wallClock(now, timezone);
  const morning = (day: Date) =>
    set(day, { hours: MORNING_HOUR, minutes: 0, seconds: 0, milliseconds: 0 });

  switch (choice) {
    case "1h":
      return toSeconds(now) + HOUR_SECONDS;
    case "tomorrow":
      return fromWallClock(morning(addDays(wall, 1)), timezone);
    case "monday":
      return fromWallClock(morning(nextMonday(wall)), timezone);
    case "off":
    case "current":
      return null;
  }
}

/** When a focus chosen now ends: in one or two hours, or at the end of the person's day. */
export function focusUntil(choice: FocusChoice, now: Date, timezone: string): number {
  switch (choice) {
    case "1h":
      return toSeconds(addHours(now, 1));
    case "2h":
      return toSeconds(addHours(now, 2));
    case "eod":
      return fromWallClock(endOfDay(wallClock(now, timezone)), timezone);
  }
}

/** The choice a focus already running reads as, by how much of it is left. */
export function focusChoiceFor(until: number | null | undefined, now: Date): FocusChoice {
  if (!until) return "2h";
  const left = until - toSeconds(now);
  if (left <= HOUR_SECONDS) return "1h";
  if (left <= 2 * HOUR_SECONDS) return "2h";
  return "eod";
}

/** Every half hour of the day as "HH:MM", the values quiet hours are stored in. */
export function halfHourValues(): string[] {
  return Array.from({ length: 48 }, (_, index) => {
    const hours = Math.floor(index / 2);
    return `${String(hours).padStart(2, "0")}:${index % 2 === 0 ? "00" : "30"}`;
  });
}

/** A stored "HH:MM" as the person reads times, in their language: 7:00 PM or 19:00. */
export function formatClockValue(value: string, timeFormat: TimeFormatType): string {
  const [hoursText = "0", minutesText = "0"] = value.split(":");
  const instant = Date.UTC(1970, 0, 1, Number(hoursText) % 24, Number(minutesText));
  return dateTimeFormatter({
    hour: timeFormat === "24-hour" ? "2-digit" : "numeric",
    minute: "2-digit",
    hourCycle: timeFormat === "24-hour" ? "h23" : "h12",
    timeZone: "UTC",
  }).format(instant);
}

/** The hour of the day in a zone, 0 to 23. */
export function hourIn(now: Date, timezone: string): number {
  return wallClock(now, timezone).getHours();
}

export type GreetingPeriod = "late" | "morning" | "afternoon" | "evening";

export function greetingPeriod(hour: number): GreetingPeriod {
  if (hour < 5) return "late";
  if (hour < 12) return "morning";
  if (hour < 17) return "afternoon";
  return "evening";
}

export type PasswordCheck = "length" | "number" | "case" | "symbol";

export const PASSWORD_MIN_LENGTH = 12;

/** Which of the four requirements a new password meets. */
export function passwordChecks(password: string): Record<PasswordCheck, boolean> {
  return {
    length: password.length >= PASSWORD_MIN_LENGTH,
    number: /\d/.test(password),
    case: /\p{Ll}/u.test(password) && /\p{Lu}/u.test(password),
    symbol: /[^\p{L}\p{N}\s]/u.test(password),
  };
}

export function passwordScore(password: string): number {
  return Object.values(passwordChecks(password)).filter(Boolean).length;
}

/** Whole days between an instant and now, never negative. */
export function daysSince(unixSeconds: number, now: Date): number {
  return Math.max(0, Math.floor((toSeconds(now) - unixSeconds) / 86400));
}

export type SignInKind =
  | "password-and-code"
  | "recovery-code"
  | "password"
  | "code-requested"
  | "single-sign-on"
  | "wrong-code"
  | "wrong-password"
  | "refused"
  | "throttled";

export type SignInTone = "ok" | "pending" | "bad";

type SignInEvent = { provider: string; outcome: string; mfaState: string };

/** What one recorded sign-in attempt was, as the person reads it in their log. */
export function describeSignIn(event: SignInEvent): { kind: SignInKind; tone: SignInTone } {
  const succeeded = event.outcome === "success";
  if (event.provider === "login_throttled") return { kind: "throttled", tone: "bad" };
  if (event.provider === "mfa") {
    if (!succeeded) return { kind: "wrong-code", tone: "bad" };
    return event.mfaState === "recovery_code"
      ? { kind: "recovery-code", tone: "ok" }
      : { kind: "password-and-code", tone: "ok" };
  }
  if (event.provider === "sso" || event.provider.startsWith("sso.")) {
    return succeeded ? { kind: "single-sign-on", tone: "ok" } : { kind: "refused", tone: "bad" };
  }
  if (!succeeded) {
    return event.outcome === "denied"
      ? { kind: "refused", tone: "bad" }
      : { kind: "wrong-password", tone: "bad" };
  }
  return event.mfaState === "challenged"
    ? { kind: "code-requested", tone: "pending" }
    : { kind: "password", tone: "ok" };
}
