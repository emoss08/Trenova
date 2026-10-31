import { LOCALE_NAMES, LOCALES, type Locale } from "@trenova/shared/i18n/generated/locales";

export type LanguageOption = {
  locale: Locale;
  /** The language in itself: what a reader of it looks for. */
  nativeName: string;
  /** The language in the interface's current language. */
  displayName: string;
  /** Every name the option answers a search to, folded for comparison. */
  searchText: string;
};

/**
 * The tag a language is named by. A language Trenova speaks once is named alone
 * ("Spanish", not "Spanish (Mexico)"). Two of the same language are told apart
 * by what differs between them: the script when they are written differently
 * (Chinese (Traditional) and Chinese (Simplified)), otherwise the region.
 */
function nameTag(locale: Locale): string {
  const { language, script } = new Intl.Locale(locale).maximize();
  const siblings = LOCALES.filter(
    (other) => other !== locale && new Intl.Locale(other).language === language,
  );
  if (siblings.length === 0) return language;
  const scriptsDiffer = siblings.some(
    (other) => new Intl.Locale(other).maximize().script !== script,
  );
  return scriptsDiffer && script ? `${language}-${script}` : locale;
}

function languageName(tag: string, inLocale: string): string | undefined {
  try {
    return new Intl.DisplayNames([inLocale], { type: "language", languageDisplay: "standard" }).of(
      tag,
    );
  } catch {
    return undefined;
  }
}

/** Lower case without accents, so "espanol" finds "Español". */
export function foldForSearch(text: string): string {
  return text
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLocaleLowerCase();
}

/**
 * Every language Trenova is translated into, each named in itself and in the
 * language the interface is in now. A search also matches the English name
 * and the locale code, so someone lost in a language they cannot read can
 * still type "English".
 */
export function languageOptions(displayLocale: Locale): LanguageOption[] {
  return LOCALES.map((locale) => {
    const tag = nameTag(locale);
    const nativeName = LOCALE_NAMES[locale];
    const displayName = languageName(tag, displayLocale) ?? nativeName;
    const englishName = languageName(tag, "en") ?? "";
    return {
      locale,
      nativeName,
      displayName,
      searchText: foldForSearch([nativeName, displayName, englishName, locale].join(" ")),
    };
  });
}

export function filterLanguageOptions(
  options: readonly LanguageOption[],
  query: string,
): LanguageOption[] {
  const needle = foldForSearch(query.trim());
  if (needle === "") return [...options];
  return options.filter((option) => option.searchText.includes(needle));
}
