import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** A duration in words as the hours band writes it: "4h 48m", "6h", "35m", "0h". */
export function formatHoursMinutes(t: TranslateFn, ms: number): string {
  const totalMinutes = Math.max(0, Math.floor(ms / 60_000));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours === 0 && minutes > 0) return t("{0}m", minutes);
  if (minutes === 0) return t("{0}h", hours);
  return t("{0}h {1}m", hours, minutes);
}

/** A limit in whole hours ("11"), for "of 11h driving". */
export function limitHours(ms: number): number {
  return Math.round(ms / 3_600_000);
}
