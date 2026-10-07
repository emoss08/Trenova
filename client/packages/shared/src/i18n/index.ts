export {
  type CatalogBundle,
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
  formatRelativeTime,
  intlLocale,
} from "@trenova/shared/i18n/format";
export { I18nProvider } from "@trenova/shared/i18n/provider";
export {
  afterCatalogs,
  getLocale,
  hasTranslation,
  loadCatalog,
  lookupIn,
  requireCatalog,
  setLocale,
  subscribe,
  translate,
  translateIn,
  whenCatalogsReady,
} from "@trenova/shared/i18n/runtime";
export {
  type RichTag,
  type RichTags,
  type RichTranslateFn,
  renderRich,
  translateRich,
  useRichT,
} from "@trenova/shared/i18n/rich";
export { type TranslateFn, useLocale, useT } from "@trenova/shared/i18n/use-t";
