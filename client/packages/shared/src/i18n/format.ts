// format.ts routes every Intl formatter through the active locale.
//
// Before this existed the client hardcoded "en-US" in a dozen formatters, which meant a
// Spanish or Chinese user still saw 1,234.56 and March 3, 2026. Numbers, dates and
// currency are as much a part of a translation as the words are.
import { DEFAULT_LOCALE, type Locale } from "@trenova/shared/i18n/generated/locales";
import { getLocale } from "@trenova/shared/i18n/runtime";

// The product's English is US English, so `en` resolves to en-US rather than the bare tag.
// The others are already region- or script-qualified.
const INTL_TAGS: Record<Locale, string> = {
  en: "en-US",
  es: "es-419",
  "zh-TW": "zh-TW",
  "zh-CN": "zh-CN",
};

export function intlLocale(locale?: Locale): string {
  return INTL_TAGS[locale ?? getLocale()] ?? INTL_TAGS[DEFAULT_LOCALE];
}

// Building an Intl formatter costs far more than using one, and a table cell
// formats on every render, so formatters are kept per locale and options. The
// options are plain literals, so their JSON is a faithful key; an undefined
// option is dropped from the key exactly as Intl ignores it.
const FORMATTER_CACHE_LIMIT = 256;

function cachedFormatter<T>(
  cache: Map<string, T>,
  locale: string,
  options: object | undefined,
  create: () => T,
): T {
  const key = `${locale}|${options ? JSON.stringify(options) : ""}`;
  let formatter = cache.get(key);
  if (formatter === undefined) {
    if (cache.size >= FORMATTER_CACHE_LIMIT) {
      cache.delete(cache.keys().next().value as string);
    }
    formatter = create();
    cache.set(key, formatter);
  }
  return formatter;
}

const numberFormatters = new Map<string, Intl.NumberFormat>();
const dateTimeFormatters = new Map<string, Intl.DateTimeFormat>();
const listFormatters = new Map<string, Intl.ListFormat>();
const relativeTimeFormatters = new Map<string, Intl.RelativeTimeFormat>();

export function numberFormatter(
  options?: Intl.NumberFormatOptions,
  locale: string = intlLocale(),
): Intl.NumberFormat {
  return cachedFormatter(
    numberFormatters,
    locale,
    options,
    () => new Intl.NumberFormat(locale, options),
  );
}

export function dateTimeFormatter(
  options?: Intl.DateTimeFormatOptions,
  locale: string = intlLocale(),
): Intl.DateTimeFormat {
  return cachedFormatter(
    dateTimeFormatters,
    locale,
    options,
    () => new Intl.DateTimeFormat(locale, options),
  );
}

export function formatNumber(value: number, options?: Intl.NumberFormatOptions): string {
  return numberFormatter(options).format(value);
}

export function formatList(
  items: readonly string[],
  type: Intl.ListFormatType = "conjunction",
): string {
  const locale = intlLocale();
  const options: Intl.ListFormatOptions = { style: "long", type };
  return cachedFormatter(
    listFormatters,
    locale,
    options,
    () => new Intl.ListFormat(locale, options),
  ).format(items);
}

const RELATIVE_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 31_536_000],
  ["month", 2_592_000],
  ["week", 604_800],
  ["day", 86_400],
  ["hour", 3_600],
  ["minute", 60],
];

export function formatRelativeTime(deltaSeconds: number): string {
  const locale = intlLocale();
  const options: Intl.RelativeTimeFormatOptions = { numeric: "auto" };
  const formatter = cachedFormatter(
    relativeTimeFormatters,
    locale,
    options,
    () => new Intl.RelativeTimeFormat(locale, options),
  );

  for (const [unit, seconds] of RELATIVE_UNITS) {
    if (Math.abs(deltaSeconds) >= seconds) {
      return formatter.format(Math.round(deltaSeconds / seconds), unit);
    }
  }

  return formatter.format(Math.round(deltaSeconds), "second");
}

const ordinalRules = new Map<string, Intl.PluralRules>();
const ENGLISH_ORDINAL_SUFFIX: Partial<Record<Intl.LDMLPluralRule, string>> = {
  one: "st",
  two: "nd",
  few: "rd",
  other: "th",
};

/**
 * formatOrdinal writes a position the way the reader's language does inside a sentence:
 * "15th" in English, where the suffix carries the meaning. Spanish and Chinese put the
 * marker in the sentence itself ("el día 15", "15 日"), so the message holds it and the
 * number stays plain.
 */
export function formatOrdinal(value: number): string {
  const locale = getLocale();
  if (locale !== DEFAULT_LOCALE) return formatNumber(value);

  const tag = intlLocale(locale);
  const rule = cachedFormatter(
    ordinalRules,
    tag,
    { type: "ordinal" },
    () => new Intl.PluralRules(tag, { type: "ordinal" }),
  ).select(value);
  return `${formatNumber(value)}${ENGLISH_ORDINAL_SUFFIX[rule] ?? ENGLISH_ORDINAL_SUFFIX.other}`;
}
