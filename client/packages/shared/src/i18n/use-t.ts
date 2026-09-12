import { getLocale, subscribe, translateIn } from "@trenova/shared/i18n/runtime";
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import { useCallback, useSyncExternalStore } from "react";

export type TranslateFn = (message: string | null | undefined, ...args: unknown[]) => string;

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

  // Bound to the rendered locale, which also gives the returned function a new identity per
  // language — downstream useMemo over `t` (table column definitions, zod schemas) then
  // rebuilds instead of holding on to text in the previous language.
  return useCallback(
    (message: string | null | undefined, ...args: unknown[]) =>
      translateIn(locale, message, ...args),
    [locale],
  );
}
