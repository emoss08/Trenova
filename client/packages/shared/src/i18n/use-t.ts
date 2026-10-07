import {
  getCatalogVersion,
  getLocale,
  getTranslator,
  subscribe,
  type TranslateFn,
} from "@trenova/shared/i18n/runtime";
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import { useSyncExternalStore } from "react";

export type { TranslateFn };

export function useLocale(): Locale {
  return useSyncExternalStore(subscribe, getLocale, getLocale);
}

/**
 * useT returns the translate function bound to the locale React is rendering with, and
 * re-renders the component when the locale changes or a catalog bundle arrives. Outside
 * React, import `translate` from the runtime directly — it reads the same active locale.
 *
 * The function gets a new identity per language and only then, so downstream useMemo over
 * `t` (table column definitions, zod schemas) rebuilds instead of holding on to text in the
 * previous language, while effects keyed on `t` do not re-run when a route's strings land.
 */
export function useT(): TranslateFn {
  useSyncExternalStore(subscribe, getCatalogVersion, getCatalogVersion);
  return useSyncExternalStore(subscribe, getTranslator, getTranslator);
}
