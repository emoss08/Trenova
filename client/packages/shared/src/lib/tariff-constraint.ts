// tariff-constraint.ts is what a form knows about an input's `where` clause: the bounds in words
// for the field's help text, and whether a value typed so far is inside them, by the crate's own
// rule (formula-language.md, "Input constraints"): the minimum is inclusive, the maximum only
// when `maxInclusive`, a list is matched by value, and a `List<T>` input is bounded item by item.
// A range orders its values as the language does: numbers, money and quantities, dates,
// date-times and times in time order, text by Unicode code point and an ordered enum's variants
// (`FreightClass`, `Weekday`) by their place in it.
// The module stays the authority: `tariff.check` refuses every saved value outside a constraint.

import { formatList } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatTariffValue, type TariffValueFormatOptions } from "@trenova/shared/lib/tariff-value";
import {
  isTariffDate,
  isTariffNumber,
  isTariffQuantity,
  isTariffTime,
  normalizeTariffTime,
} from "@trenova/shared/rules/tariff-wire";
import type { TariffConstraint, TariffType, TariffWireValue } from "@trenova/shared/types/tariff";

/** `-12.50` as a sign, digits and a scale, so two decimals compare without a float. */
function scaled(text: string): { value: bigint; scale: number } | null {
  const match = /^(-)?(\d+)(?:\.(\d+))?$/.exec(text.trim());
  if (!match) return null;
  const fraction = match[3] ?? "";
  const digits = BigInt(`${match[2]}${fraction}`);
  return { value: match[1] ? -digits : digits, scale: fraction.length };
}

/** Compares two exact decimal texts: negative, zero or positive; null when either is not one. */
export function compareDecimalText(a: string, b: string): number | null {
  const left = scaled(a);
  const right = scaled(b);
  if (!left || !right) return null;
  const scale = Math.max(left.scale, right.scale);
  const x = left.value * 10n ** BigInt(scale - left.scale);
  const y = right.value * 10n ** BigInt(scale - right.scale);
  return x === y ? 0 : x < y ? -1 : 1;
}

/** A date or date-time's instant, so values with different offsets compare as moments. */
function instant(text: string): number | null {
  const at = Date.parse(text.length === 10 ? `${text}T00:00:00Z` : text);
  return Number.isNaN(at) ? null : at;
}

/**
 * Orders two texts as Tariff does: by Unicode code point, character by character, a text before
 * every longer text it starts. Never by locale; JavaScript's `<` compares UTF-16 code units,
 * which puts a character past the Basic Multilingual Plane before `U+E000`–`U+FFFF`.
 */
export function compareCodePoints(a: string, b: string): number {
  let at = 0;
  while (at < a.length && at < b.length) {
    const left = a.codePointAt(at) ?? 0;
    const right = b.codePointAt(at) ?? 0;
    if (left !== right) return left < right ? -1 : 1;
    at += left > 0xffff ? 2 : 1;
  }
  return a.length === b.length ? 0 : a.length < b.length ? -1 : 1;
}

function scalar(type: TariffType): TariffType {
  return type.kind === "optional" ? scalar(type.inner) : type;
}

/**
 * Orders two wire values of one ordered type, the value's declared `type`: numbers, quantities in
 * the same unit, dates and date-times as moments, times of day, text by code point and an
 * ordered enum's variants by their place in it. Null when they cannot be ordered here (a quantity
 * in another unit of the same dimension, which only the module converts exactly; the variants of
 * an enum with no order), so the caller leaves the value alone.
 */
export function compareTariffValues(
  a: TariffWireValue,
  b: TariffWireValue,
  type: TariffType,
): number | null {
  if (isTariffNumber(a) && isTariffNumber(b)) return compareDecimalText(a.$n, b.$n);
  if (isTariffQuantity(a) && isTariffQuantity(b)) {
    return a.u === b.u ? compareDecimalText(a.$q, b.$q) : null;
  }
  if (isTariffDate(a) && isTariffDate(b)) {
    const left = instant(a.$date);
    const right = instant(b.$date);
    return left === null || right === null ? null : Math.sign(left - right);
  }
  if (isTariffTime(a) && isTariffTime(b)) {
    const left = normalizeTariffTime(a.$time);
    const right = normalizeTariffTime(b.$time);
    return left === null || right === null ? null : compareCodePoints(left, right);
  }
  if (typeof a === "string" && typeof b === "string") {
    const declared = scalar(type);
    if (declared.kind === "text") return compareCodePoints(a, b);
    if (declared.kind === "enum" && declared.ordered) {
      const left = declared.variants.indexOf(a);
      const right = declared.variants.indexOf(b);
      return left < 0 || right < 0 ? null : Math.sign(left - right);
    }
  }
  return null;
}

/** Whether two wire values are the same value: decimals by value, lists and records deeply. */
export function sameTariffValue(a: TariffWireValue, b: TariffWireValue): boolean {
  if (
    (isTariffNumber(a) && isTariffNumber(b)) ||
    (isTariffQuantity(a) && isTariffQuantity(b)) ||
    (isTariffDate(a) && isTariffDate(b)) ||
    (isTariffTime(a) && isTariffTime(b))
  ) {
    return compareTariffValues(a, b, { kind: "unknown" }) === 0;
  }
  if (Array.isArray(a) || Array.isArray(b)) {
    return (
      Array.isArray(a) &&
      Array.isArray(b) &&
      a.length === b.length &&
      a.every((item, index) => sameTariffValue(item, b[index] as TariffWireValue))
    );
  }
  if (a !== null && b !== null && typeof a === "object" && typeof b === "object") {
    const keys = Object.keys(a);
    return (
      keys.length === Object.keys(b).length &&
      keys.every(
        (key) =>
          key in b &&
          sameTariffValue(
            (a as Record<string, TariffWireValue>)[key] as TariffWireValue,
            (b as Record<string, TariffWireValue>)[key] as TariffWireValue,
          ),
      )
    );
  }
  return a === b;
}

/**
 * Whether one value of `type` (an item and the item type, for a list input) is outside the
 * constraint. `none` always passes, and a value that cannot be compared here (another unit) is
 * left to the server's check.
 */
export function outsideTariffConstraint(
  constraint: TariffConstraint,
  value: TariffWireValue,
  type: TariffType,
): boolean {
  if (value === null) return false;
  if ("oneOf" in constraint) {
    return !constraint.oneOf.some((allowed) => sameTariffValue(allowed, value));
  }
  const { min, max, maxInclusive } = constraint.range;
  if (min !== null) {
    const order = compareTariffValues(value, min, type);
    if (order !== null && order < 0) return true;
  }
  if (max !== null) {
    const order = compareTariffValues(value, max, type);
    if (order !== null && (maxInclusive ? order > 0 : order >= 0)) return true;
  }
  return false;
}

/** Whether a declared type is a list, whose constraint bounds each of its items. */
export function isListType(type: TariffType): boolean {
  return (type.kind === "optional" ? type.inner : type).kind === "list";
}

/**
 * The index of each item outside the constraint for a list input, or `[-1]` when a single value
 * is outside it; empty when everything fits. Values of the wrong shape are left to the type check.
 */
export function tariffConstraintViolations(
  constraint: TariffConstraint,
  type: TariffType,
  value: TariffWireValue | undefined,
): number[] {
  if (value === undefined || value === null) return [];
  const declared = scalar(type);
  if (declared.kind === "list") {
    if (!Array.isArray(value)) return [];
    return value.flatMap((item, index) =>
      outsideTariffConstraint(constraint, item, declared.item) ? [index] : [],
    );
  }
  return outsideTariffConstraint(constraint, value, declared) ? [-1] : [];
}

/**
 * The variants of an ordered enum input that its `where` range allows, in order (`Class50` to
 * `Class100` of `FreightClass`), which a field offers as its choices; null for any other input
 * or constraint.
 */
export function orderedEnumChoices(
  constraint: TariffConstraint | null | undefined,
  type: TariffType,
): string[] | null {
  if (!constraint || !("range" in constraint)) return null;
  const declared = scalar(type);
  if (declared.kind !== "enum" || !declared.ordered) return null;
  return declared.variants.filter(
    (variant) => !outsideTariffConstraint(constraint, variant, declared),
  );
}

/**
 * The bounds in words, for a field's help text: "From $0.50/mi to $15.00/mi", "One of Standard,
 * Guaranteed". A list input's constraint bounds each value, and says so.
 */
export function describeTariffConstraint(
  constraint: TariffConstraint,
  t: TranslateFn,
  options: TariffValueFormatOptions & { each?: boolean } = {},
): string {
  const show = (value: TariffWireValue) => formatTariffValue(value, t, options);
  const each = options.each ?? false;
  if ("oneOf" in constraint) {
    const list = formatList(constraint.oneOf.map(show), "unit");
    return each ? t("Each one of {0}", list) : t("One of {0}", list);
  }
  const { min, max, maxInclusive } = constraint.range;
  if (min !== null && max !== null) {
    if (maxInclusive) {
      return each
        ? t("Each from {0} to {1}", show(min), show(max))
        : t("From {0} to {1}", show(min), show(max));
    }
    return each
      ? t("Each at least {0} and below {1}", show(min), show(max))
      : t("At least {0} and below {1}", show(min), show(max));
  }
  if (min !== null) return each ? t("Each at least {0}", show(min)) : t("At least {0}", show(min));
  if (max !== null) {
    if (maxInclusive) {
      return each ? t("Each at most {0}", show(max)) : t("At most {0}", show(max));
    }
    return each ? t("Each below {0}", show(max)) : t("Below {0}", show(max));
  }
  return t("Any value");
}

/** The field error a value outside the constraint shows: the bounds it must keep to. */
export function tariffConstraintMessage(
  constraint: TariffConstraint,
  t: TranslateFn,
  options: TariffValueFormatOptions = {},
): string {
  const show = (value: TariffWireValue) => formatTariffValue(value, t, options);
  if ("oneOf" in constraint) {
    return t("Must be one of {0}", formatList(constraint.oneOf.map(show), "unit"));
  }
  const { min, max, maxInclusive } = constraint.range;
  if (min !== null && max !== null) {
    return maxInclusive
      ? t("Must be from {0} to {1}", show(min), show(max))
      : t("Must be at least {0} and below {1}", show(min), show(max));
  }
  if (min !== null) return t("Must be at least {0}", show(min));
  if (max !== null) {
    return maxInclusive ? t("Must be at most {0}", show(max)) : t("Must be below {0}", show(max));
  }
  return t("Must fit the input's bounds");
}
