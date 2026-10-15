import { numberFormatter } from "@trenova/shared/i18n/format";

const AMOUNT_PATTERN = /^(\d+)(?:\.(\d{1,2}))?$/;

/**
 * Minor units (cents) from an amount a person typed, read from the digits
 * rather than through a float, so 0.29 is 29 and not 28.999…. Returns null
 * for anything that is not a plain non-negative amount with at most two
 * decimal places; rounding a third decimal away would change the money
 * without anyone deciding to.
 */
export function parseAmountToMinor(value: string): number | null {
  const match = AMOUNT_PATTERN.exec(value.trim());
  if (!match) {
    return null;
  }
  const whole = Number(match[1]);
  const fraction = Number((match[2] ?? "").padEnd(2, "0"));
  const minor = whole * 100 + fraction;

  return Number.isSafeInteger(minor) ? minor : null;
}

/**
 * Renders a minor-unit amount (cents) as money in the reader's language.
 * Integer arithmetic all the way to the formatter: dividing by 100 is the
 * last thing that happens, not the first.
 */
export function formatMinor(minor: number, currency = "USD"): string {
  return numberFormatter({
    style: "currency",
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(minor / 100);
}
