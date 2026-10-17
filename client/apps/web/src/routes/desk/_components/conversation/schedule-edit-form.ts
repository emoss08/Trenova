import type { ConversationSchedule } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { z } from "zod";

/** The most a scheduled request may ask, as the server keeps it. */
export const MAX_SCHEDULE_PROMPT = 2000;

/**
 * How often a schedule can run, as the server accepts an edit: every day,
 * every weekday, or one day of the week. They are the cadences a message can
 * ask for, so every schedule can be edited.
 */
export const SCHEDULE_REPEATS = [
  "day",
  "weekday",
  "monday",
  "tuesday",
  "wednesday",
  "thursday",
  "friday",
  "saturday",
  "sunday",
] as const;

export type ScheduleRepeat = (typeof SCHEDULE_REPEATS)[number];

export type ScheduleEditValues = {
  prompt: string;
  repeat: ScheduleRepeat;
  time: string;
};

/** The cron day-of-week field each single day is written as. */
const DAY_FIELDS: Record<string, ScheduleRepeat> = {
  "0": "sunday",
  "7": "sunday",
  "1": "monday",
  "2": "tuesday",
  "3": "wednesday",
  "4": "thursday",
  "5": "friday",
  "6": "saturday",
};

const CRON_NUMBER = /^\d{1,2}$/;

const pad = (value: number) => String(value).padStart(2, "0");

/**
 * A schedule as its editor opens on it: what it asks, how often and the time
 * on a 24-hour clock, read from its cron. A cron the editor does not offer
 * opens on every day at the time it names, or 8:00, so saving it is a choice.
 */
export function scheduleEditValues(
  schedule: Pick<ConversationSchedule, "prompt" | "cronExpression">,
): ScheduleEditValues {
  const fields = schedule.cronExpression.trim().split(/\s+/);
  const [minuteField = "", hourField = "", , , dayOfWeek = ""] = fields;
  const timed =
    fields.length === 5 &&
    CRON_NUMBER.test(minuteField) &&
    CRON_NUMBER.test(hourField) &&
    Number(hourField) < 24 &&
    Number(minuteField) < 60;
  const time = timed ? `${pad(Number(hourField))}:${pad(Number(minuteField))}` : "08:00";
  const repeat: ScheduleRepeat =
    dayOfWeek === "*" ? "day" : dayOfWeek === "1-5" ? "weekday" : (DAY_FIELDS[dayOfWeek] ?? "day");

  return { prompt: schedule.prompt, repeat, time };
}

/** What an edit must hold before it is sent; the server checks it again. */
export function scheduleEditSchema(t: TranslateFn) {
  return z.object({
    prompt: z
      .string()
      .trim()
      .min(1, t("Say what to ask on each run"))
      .max(MAX_SCHEDULE_PROMPT, t("A scheduled request cannot be longer than 2000 characters")),
    repeat: z.enum(SCHEDULE_REPEATS, { error: () => t("Choose how often it runs") }),
    time: z.string().regex(/^([01]?\d|2[0-3]):[0-5]\d$/, t("Enter a time of day, such as 08:00")),
  });
}
