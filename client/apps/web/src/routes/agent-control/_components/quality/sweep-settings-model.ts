/** The hours a sweep is usually started at; another hour set elsewhere is added. */
export const START_HOUR_PRESETS = [0, 2, 4, 6] as const;
export const CASES_PER_AGENT_PRESETS = [20, 40, 100] as const;
export const RERUN_DAYS_PRESETS = [7, 14, 30] as const;
export const JUDGE_SHARE_PRESETS = [10, 25, 50, 100] as const;
export const THRESHOLD_PRESETS = [3, 5, 10] as const;
export const MIN_CASES_PRESETS = [3, 5, 10] as const;
/** Budgets in cents: fifty cents, a dollar and five a night; ten, twenty and a hundred a month. */
export const NIGHTLY_BUDGET_PRESETS = [50, 100, 500] as const;
export const MONTHLY_BUDGET_PRESETS = [1000, 2000, 10000] as const;

/** An hour of the day on the 24-hour clock, as the sweep's start reads. */
export function hourLabel(hour: number): string {
  return `${String(hour).padStart(2, "0")}:00`;
}
