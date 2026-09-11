import { getLocale, subscribe, translate } from "@trenova/shared/i18n/runtime";
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import { useCallback, useSyncExternalStore } from "react";

export type TranslateFn = (message: string, ...args: unknown[]) => string;

export function useLocale(): Locale {
  return useSyncExternalStore(subscribe, getLocale, getLocale);
}

/**
 * useT returns the translate function rebound whenever the locale changes, so components
 * re-render on a language switch. Outside React, import `translate` from the runtime
 * directly — it reads the same active locale.
 */
export function useT(): TranslateFn {
  const locale = useLocale();

  return useCallback(
    (message: string, ...args: unknown[]) => translate(message, ...args),
    // The identity of `translate` never changes; `locale` is the real dependency, and
    // rebinding on it is what makes consumers re-render when the language changes.
    [locale],
  );
}
