import { intlLocale, numberFormatter } from "../i18n/format";
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

const symbols = new Map<string, string>();

/**
 * The mark a currency is written with in the reader's locale ("$" for US dollars), for a
 * field prefix. A form draws one per money field on every keystroke, so each locale and
 * currency is worked out once.
 */
export function currencySymbol(currency?: string | null): string {
  const code = currency || "USD";
  const key = `${intlLocale()}|${code}`;
  let symbol = symbols.get(key);
  if (symbol === undefined) {
    symbol =
      numberFormatter({ style: "currency", currency: code })
        .formatToParts(0)
        .find((part) => part.type === "currency")?.value ?? "";
    symbols.set(key, symbol);
  }
  return symbol;
}

export type AmountSeparators = { decimal: string; group: string };

const separators = new Map<string, AmountSeparators>();

/**
 * How the reader's locale writes an amount: the mark before the cents and the mark between
 * thousands ("." and "," in English, "," and "." in Spanish). A money field types and shows
 * amounts with these, so a figure reads in the field as it does everywhere else on screen.
 */
export function amountSeparators(): AmountSeparators {
  const locale = intlLocale();
  let found = separators.get(locale);
  if (!found) {
    const parts = numberFormatter({ minimumFractionDigits: 2 }).formatToParts(1234567.89);
    found = {
      decimal: parts.find((part) => part.type === "decimal")?.value ?? ".",
      group: parts.find((part) => part.type === "group")?.value ?? "",
    };
    separators.set(locale, found);
  }
  return found;
}

const AMOUNT_TEXT = /^(-?)(\d*)(?:\.(\d{0,2}))?/;

/**
 * Minor units from the plain text of a field being typed in ("12.5", "-3", "", "."), read
 * from its digits and never through a float. Text with no digits yet is zero, and digits
 * past the second decimal are dropped, as the field itself allows none.
 */
export function amountTextToMinor(text: string): number {
  const match = AMOUNT_TEXT.exec(text.trim());
  if (!match) {
    return 0;
  }
  const [, sign, whole = "", fraction = ""] = match;
  const minor = Number(whole || "0") * 100 + Number(fraction.padEnd(2, "0") || "0");
  if (!Number.isSafeInteger(minor)) {
    return 0;
  }
  return sign && minor !== 0 ? -minor : minor;
}

/** A minor-unit amount as the plain text a field holds ("12.50", "-3.00"), by its digits. */
export function minorToAmountText(minor: number): string {
  const magnitude = Math.abs(Math.trunc(minor));
  const whole = Math.trunc(magnitude / 100);
  const fraction = String(magnitude % 100).padStart(2, "0");
  return `${minor < 0 ? "-" : ""}${whole}.${fraction}`; // i18n-ignore: a number's digits
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
