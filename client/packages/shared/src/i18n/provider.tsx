import { DEFAULT_LOCALE, isLocale, type Locale } from "@trenova/shared/i18n/generated/locales";
import { getLocale, loadCatalog, setLocale, subscribe } from "@trenova/shared/i18n/runtime";
import { type ReactNode, useEffect, useState, useSyncExternalStore } from "react";

const STORAGE_KEY = "trenova.locale";

export function readStoredLocale(): Locale | null {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    return stored !== null && isLocale(stored) ? stored : null;
  } catch {
    // Private browsing and blocked site data both throw here. A remembered language is a
    // convenience; losing it must never stop the app rendering.
    return null;
  }
}

export function storeLocale(locale: Locale): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, locale);
  } catch {
    // See readStoredLocale.
  }
}

function fromNavigator(): Locale | null {
  if (typeof navigator === "undefined") return null;

  for (const tag of navigator.languages ?? [navigator.language]) {
    if (isLocale(tag)) return tag;

    // A browser reporting zh-Hant / zh-Hans / zh-HK still has a clear written form.
    const lower = tag.toLowerCase();
    if (lower.startsWith("zh")) {
      if (/hant|tw|hk|mo/.test(lower)) return "zh-TW";
      return "zh-CN";
    }
    const base = lower.split("-")[0];
    if (isLocale(base)) return base;
  }

  return null;
}

/**
 * resolveInitialLocale prefers what the signed-in user chose, because that preference is
 * stored server-side and is what their emails and documents already use. The browser's
 * languages are only a guess for someone who has not chosen yet.
 */
export function resolveInitialLocale(userLocale?: string | null): Locale {
  if (userLocale != null && isLocale(userLocale)) return userLocale;
  return readStoredLocale() ?? fromNavigator() ?? DEFAULT_LOCALE;
}

type I18nProviderProps = {
  children: ReactNode;
  userLocale?: string | null;
  fallback?: ReactNode;
};

export function I18nProvider({ children, userLocale, fallback = null }: I18nProviderProps) {
  const active = useSyncExternalStore(subscribe, getLocale, getLocale);
  const [ready, setReady] = useState(() => resolveInitialLocale(userLocale) === DEFAULT_LOCALE);

  useEffect(() => {
    const target = resolveInitialLocale(userLocale);
    if (target === active && ready) return;

    let cancelled = false;
    loadCatalog(target)
      .then(() => {
        if (cancelled) return undefined;
        // Remembered so the login screen, which has no user to read a preference from,
        // still comes up in the language this browser last used.
        storeLocale(target);
        return setLocale(target);
      })
      .catch(() => {
        // A catalog that fails to load leaves the app in English rather than blank.
      })
      .finally(() => {
        if (!cancelled) setReady(true);
      });

    return () => {
      cancelled = true;
    };
  }, [userLocale, active, ready]);

  if (!ready) return <>{fallback}</>;

  return <>{children}</>;
}
