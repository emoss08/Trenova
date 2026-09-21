/**
 * How many clean approvals in a row earn a tool its next tier. A short list
 * keeps the choice legible; a value set some other way still shows up as the
 * pressed option rather than leaving the control blank.
 */
export const PROMOTION_THRESHOLD_PRESETS = [5, 10, 25, 50] as const;

export type ThresholdOption = { value: number; label: string };

export function promotionThresholdOptions(current: number): ThresholdOption[] {
  const values = new Set<number>(PROMOTION_THRESHOLD_PRESETS);
  if (Number.isInteger(current) && current > 0) {
    values.add(current);
  }

  return [...values].sort((a, b) => a - b).map((value) => ({ value, label: String(value) }));
}
