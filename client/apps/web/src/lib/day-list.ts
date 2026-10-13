/**
 * Reads a list of whole days typed as "1, 15, 30" into numbers, sorted and
 * without repeats. Returns null when any entry is not a whole number, so a
 * typo is reported rather than silently dropped.
 */
export function parseDayList(text: string): number[] | null {
  const entries = text
    .split(/[,\s]+/)
    .map((entry) => entry.trim())
    .filter((entry) => entry !== "");
  const days: number[] = [];
  for (const entry of entries) {
    if (!/^\d+$/.test(entry)) {
      return null;
    }
    days.push(Number(entry));
  }
  return [...new Set(days)].sort((a, b) => a - b);
}

/** Writes a day list back the way a person types it. */
export function formatDayList(days: readonly number[] | null | undefined): string {
  return (days ?? []).join(", ");
}
