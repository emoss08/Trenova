// intl-tags.ts names the Intl locale each catalog locale formats with. format.ts uses it for
// every formatter and format-message.ts for a plural's count, so a number reads the same in
// a sentence as it does in a table cell.
import { DEFAULT_LOCALE, type Locale } from "@trenova/shared/i18n/generated/locales";

// The product's English is US English, so `en` resolves to en-US rather than the bare tag.
// The others are already region- or script-qualified.
const INTL_TAGS: Record<Locale, string> = {
  en: "en-US",
  es: "es-419",
  "zh-TW": "zh-TW",
  "zh-CN": "zh-CN",
};

export function intlTag(locale: Locale): string {
  return INTL_TAGS[locale] ?? INTL_TAGS[DEFAULT_LOCALE];
}
