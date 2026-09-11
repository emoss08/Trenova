// format.ts routes every Intl formatter through the active locale.
//
// Before this existed the client hardcoded "en-US" in a dozen formatters, which meant a
// Spanish or Chinese user still saw 1,234.56 and March 3, 2026. Numbers, dates and
// currency are as much a part of a translation as the words are.
import {
  DEFAULT_LOCALE,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";
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

export function formatNumber(value: number, options?: Intl.NumberFormatOptions): string {
  return new Intl.NumberFormat(intlLocale(), options).format(value);
}

export function formatPercent(value: number, fractionDigits = 1): string {
  return new Intl.NumberFormat(intlLocale(), {
    style: "percent",
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  }).format(value);
}

export function formatList(
  items: readonly string[],
  type: Intl.ListFormatType = "conjunction",
): string {
  return new Intl.ListFormat(intlLocale(), { style: "long", type }).format(items);
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
  const formatter = new Intl.RelativeTimeFormat(intlLocale(), { numeric: "auto" });

  for (const [unit, seconds] of RELATIVE_UNITS) {
    if (Math.abs(deltaSeconds) >= seconds) {
      return formatter.format(Math.round(deltaSeconds / seconds), unit);
    }
  }

  return formatter.format(Math.round(deltaSeconds), "second");
}
