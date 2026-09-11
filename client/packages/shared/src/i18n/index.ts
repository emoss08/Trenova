export {
  DEFAULT_LOCALE,
  isLocale,
  LOCALE_NAMES,
  LOCALES,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";
export { formatMessage } from "@trenova/shared/i18n/format-message";
export {
  formatList,
  formatNumber,
  formatPercent,
  formatRelativeTime,
  intlLocale,
} from "@trenova/shared/i18n/format";
export { I18nProvider } from "@trenova/shared/i18n/provider";
export {
  getLocale,
  hasTranslation,
  loadCatalog,
  setLocale,
  subscribe,
  translate,
} from "@trenova/shared/i18n/runtime";
export { type TranslateFn, useLocale, useT } from "@trenova/shared/i18n/use-t";
