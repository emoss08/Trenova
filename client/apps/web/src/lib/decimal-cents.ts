/** Dollars as the Decimal scalar carries them, from the cents a money field holds. */
export function centsToDecimal(cents: number): string {
  return (Math.round(cents) / 100).toFixed(2);
}

/** The cents a money field holds, from dollars as the Decimal scalar carries them. */
export function decimalToCents(value: string): number {
  const parsed = Number(value);

  return Number.isFinite(parsed) ? Math.round(parsed * 100) : 0;
}
