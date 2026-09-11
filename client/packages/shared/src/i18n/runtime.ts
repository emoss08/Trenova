// runtime.ts holds the active locale and its catalog outside React.
//
// This mirrors the preference mirror in lib/date.ts, and for the same reason: a great deal
// of translatable text is produced where no hook can run — zod schemas, TanStack Table
// column definitions, route loaders, toast calls in event handlers. Threading a locale
// through all of those is the thing every naive i18n setup gets wrong, so `translate` reads
// the active locale from here instead.
//
// React subscribes through useT(); this module is the single source it subscribes to.
import { formatMessage } from "@trenova/shared/i18n/format-message";
import {
  CATALOG_LOADERS,
  DEFAULT_LOCALE,
  isLocale,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";

type Listener = () => void;

let activeLocale: Locale = DEFAULT_LOCALE;
let activeMessages: Record<string, string> = {};
const listeners = new Set<Listener>();

const loaded = new Map<Locale, Record<string, string>>();

function notify(): void {
  for (const listener of listeners) listener();
}

export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getLocale(): Locale {
  return activeLocale;
}

export async function loadCatalog(locale: Locale): Promise<Record<string, string>> {
  const cached = loaded.get(locale);
  if (cached !== undefined) return cached;

  // English needs no catalog: the source string is the key, so lookups fall through to it.
  if (locale === DEFAULT_LOCALE) {
    loaded.set(locale, {});
    return {};
  }

  const messages = await CATALOG_LOADERS[locale]();
  loaded.set(locale, messages);
  return messages;
}

export async function setLocale(locale: Locale): Promise<void> {
  if (!isLocale(locale)) return;

  const messages = await loadCatalog(locale);
  activeLocale = locale;
  activeMessages = messages;

  if (typeof document !== "undefined") {
    document.documentElement.lang = locale;
  }

  notify();
}

/**
 * translate looks a message up by its English source text. A missing entry falls back to
 * that source rather than rendering a key, so an untranslated string is merely English —
 * never `shipment.header.title` in front of a customer.
 */
export function translate(message: string, ...args: unknown[]): string {
  return translateIn(activeLocale, message, ...args);
}

/**
 * translateIn renders in an explicitly named locale. useT() binds this to the locale React
 * has rendered with, so a language switch mid-render cannot produce a component whose text
 * is half one language and half the other.
 */
export function translateIn(locale: Locale, message: string, ...args: unknown[]): string {
  if (message === "") return "";

  const messages = locale === activeLocale ? activeMessages : (loaded.get(locale) ?? {});
  const translated = messages[message] ?? message;
  return formatMessage(locale, translated, args);
}

export function hasTranslation(message: string): boolean {
  return message in activeMessages;
}
