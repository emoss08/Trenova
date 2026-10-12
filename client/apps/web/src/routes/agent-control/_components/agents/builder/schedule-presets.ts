import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export type SchedulePreset = { cron: string; label: string; says: string };

/** The schedules offered as one click, each as the chip names it and as a sentence. */
export function schedulePresets(t: TranslateFn): SchedulePreset[] {
  return [
    { cron: "0 9 * * 5", label: t("Fridays"), says: t("Every Friday at 9:00 AM") },
    { cron: "0 6 * * 1-5", label: t("Weekday mornings"), says: t("Every weekday at 6:00 AM") },
    { cron: "0 * * * *", label: t("Every hour"), says: t("At the top of every hour") },
    { cron: "0 7 * * *", label: t("Daily"), says: t("Every day at 7:00 AM") },
    { cron: "0 8 * * 1", label: t("Mondays"), says: t("Every Monday at 8:00 AM") },
    {
      cron: "0 8 1 * *",
      label: t("First of the month"),
      says: t("The first of each month at 8:00 AM"),
    },
  ];
}

/** The preset a schedule is, or null for one written by hand. */
export function schedulePreset(cron: string, t: TranslateFn): SchedulePreset | null {
  return schedulePresets(t).find((preset) => preset.cron === cron.trim()) ?? null;
}
