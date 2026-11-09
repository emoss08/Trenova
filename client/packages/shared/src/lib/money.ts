import { numberFormatter } from "../i18n/format";
import { formatCurrency } from "./utils";

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
 * A minor-unit amount (cents) as money in the reader's locale. Integer arithmetic all the way to
 * the formatter: dividing by 100 is the last thing that happens, not the first. A record with no
 * currency reads as US dollars.
 */
export function formatMinor(minor: number, currency?: string | null): string {
  return formatCurrency(minor / 100, currency || "USD");
}

/** The mark a currency is written with in the reader's locale ("$" for US dollars), for a field prefix. */
export function currencySymbol(currency?: string | null): string {
  return (
    numberFormatter({ style: "currency", currency: currency || "USD" })
      .formatToParts(0)
      .find((part) => part.type === "currency")?.value ?? ""
  );
}

/** A minor-unit amount as a plain amount with two decimals, for a value whose currency is unknown. */
export function formatMinorAmount(minor: number): string {
  return numberFormatter({ minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(
    minor / 100,
  );
}

const MINOR_SUFFIX = /([a-z0-9])Minor$/;

/** Whether a field path holds money in minor units, by Trenova's `…Minor` naming (`balanceDueMinor`). */
export function isMinorAmountField(path: string | undefined): boolean {
  const last = path?.split(".").filter(Boolean).at(-1);
  return last !== undefined && MINOR_SUFFIX.test(last);
}

/** A field path without its minor-units suffix, so `balanceDueMinor` is named "balance due". */
export function withoutMinorSuffix(path: string): string {
  return path.replace(MINOR_SUFFIX, "$1");
}
