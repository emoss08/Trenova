// runtime.ts holds the active locale and its catalog outside React.
//
// This mirrors the preference mirror in lib/date.ts, and for the same reason: a great deal
// of translatable text is produced where no hook can run — zod schemas, TanStack Table
// column definitions, route loaders, toast calls in event handlers. Threading a locale
// through all of those is the thing every naive i18n setup gets wrong, so `translate` reads
// the active locale from here instead.
//
// A locale's strings are split into bundles (generated/locales.ts): the app shell's load at
// startup and each route folder's arrive with that folder's code. Code asks for a bundle
// with requireCatalog; the runtime keeps the set of bundles asked for and loads each one for
// whichever locale is active or being switched to, so a language switch brings every screen
// already visited along with it.
//
// React subscribes through useT(); this module is the single source it subscribes to.
import { formatMessage } from "@trenova/shared/i18n/format-message";
import {
  CATALOG_LOADERS,
  type CatalogBundle,
  DEFAULT_LOCALE,
  isLocale,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";

type Listener = () => void;

type Messages = Record<string, string>;

export type TranslateFn = (message: string | null | undefined, ...args: unknown[]) => string;

export type CatalogLoader = () => Promise<Messages>;

export type CatalogSource = Partial<Record<Locale, CatalogLoader>>;

/**
 * Everything loaded for one locale. `messages` is the one lookup table translate reads; it
 * grows in place as bundles arrive, so a screen already rendered picks up its strings on
 * the next render instead of holding a stale copy.
 */
type LocaleCatalog = {
  messages: Messages;
  bundles: Map<string, Promise<void>>;
  overlay: Messages;
  overlayLoad: Promise<void> | null;
};

let activeLocale: Locale = DEFAULT_LOCALE;
let activeMessages: Messages = {};
let activeTranslator: TranslateFn = createTranslator(DEFAULT_LOCALE);
let catalogVersion = 0;
const listeners = new Set<Listener>();

const required = new Set<string>();
const catalogs = new Map<Locale, LocaleCatalog>();
const switching = new Map<Locale, number>();
const extraSources: CatalogSource[] = [];

function notify(): void {
  for (const listener of listeners) listener();
}

function changed(): void {
  catalogVersion += 1;
  notify();
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

/**
 * getTranslator returns translate bound to the active locale. Its identity changes with the
 * locale and only then: effects and callbacks keyed on `t` must not re-run because a route's
 * strings arrived.
 */
export function getTranslator(): TranslateFn {
  return activeTranslator;
}

/**
 * getCatalogVersion changes whenever the active locale's lookup table does — a language
 * switch or a bundle landing — so subscribers can re-render to pick the new strings up.
 */
export function getCatalogVersion(): number {
  return catalogVersion;
}

function catalogFor(locale: Locale): LocaleCatalog {
  let catalog = catalogs.get(locale);
  if (catalog === undefined) {
    catalog = { messages: {}, bundles: new Map(), overlay: {}, overlayLoad: null };
    catalogs.set(locale, catalog);
  }
  return catalog;
}

function isCurrent(locale: Locale, catalog: LocaleCatalog): boolean {
  return catalogs.get(locale) === catalog;
}

function merged(locale: Locale, catalog: LocaleCatalog): void {
  if (locale === activeLocale && catalog.messages === activeMessages) changed();
}

// An edition's entries win over the app's for the same key, so its overlay is laid back on
// top every time a bundle lands underneath it.
function loadOverlay(locale: Locale, catalog: LocaleCatalog): Promise<void> {
  if (catalog.overlayLoad !== null) return catalog.overlayLoad;

  const load = Promise.all(
    extraSources.map((source) => source[locale]?.() ?? Promise.resolve({})),
  ).then(
    (overlays) => {
      if (!isCurrent(locale, catalog)) return;
      catalog.overlay = Object.assign({}, ...overlays);
      Object.assign(catalog.messages, catalog.overlay);
      merged(locale, catalog);
    },
    (err: unknown) => {
      if (isCurrent(locale, catalog)) catalog.overlayLoad = null;
      throw err;
    },
  );
  catalog.overlayLoad = load;
  return load;
}

function loadBundle(locale: Locale, catalog: LocaleCatalog, bundle: string): Promise<void> {
  const started = catalog.bundles.get(bundle);
  if (started !== undefined) return started;

  const loader = CATALOG_LOADERS[locale][bundle as CatalogBundle];
  if (loader === undefined) {
    // A route folder with no strings of its own has no bundle; asking for it is a no-op.
    const none = Promise.resolve();
    catalog.bundles.set(bundle, none);
    return none;
  }

  const load = loader().then(
    (messages) => {
      if (!isCurrent(locale, catalog)) return;
      Object.assign(catalog.messages, messages, catalog.overlay);
      merged(locale, catalog);
    },
    (err: unknown) => {
      // Forgotten so the next navigation or language switch tries again — a chunk can fail
      // to download once and succeed a moment later.
      if (isCurrent(locale, catalog)) catalog.bundles.delete(bundle);
      throw err;
    },
  );
  catalog.bundles.set(bundle, load);
  return load;
}

/**
 * settle loads every required bundle for a locale and resolves once all of them are in. A
 * bundle required while it waits (a route module evaluated mid-switch) is waited for too.
 */
async function settle(locale: Locale): Promise<Messages> {
  const catalog = catalogFor(locale);
  const awaited = new Set<Promise<void>>();

  for (;;) {
    const pending = [loadOverlay(locale, catalog)];
    for (const bundle of required) pending.push(loadBundle(locale, catalog, bundle));
    if (pending.every((load) => awaited.has(load))) return catalog.messages;

    await Promise.all(pending);
    for (const load of pending) awaited.add(load);
  }
}

// The locales a newly required bundle must load for: the one on screen and any being
// switched to, so a switch in flight cannot finish without a screen that loaded during it.
function loadingLocales(): Locale[] {
  const locales = new Set<Locale>(switching.keys());
  locales.add(activeLocale);
  locales.delete(DEFAULT_LOCALE);
  return [...locales];
}

/**
 * requireCatalog declares that code about to render needs these bundles. It is idempotent
 * and only starts downloads; await whenCatalogsReady() to wait for them. An app requires its
 * startup bundles once, and every module in a route folder requires that folder's bundle as
 * it is evaluated, which the web app's build injects.
 */
export function requireCatalog(...bundles: readonly string[]): void {
  const added = bundles.filter((bundle) => !required.has(bundle));
  if (added.length === 0) return;

  for (const bundle of added) required.add(bundle);
  for (const locale of loadingLocales()) {
    const catalog = catalogFor(locale);
    for (const bundle of added) {
      // whenCatalogsReady reports the failure to whoever waits; a fire-and-forget start
      // must not surface it as an unhandled rejection as well.
      loadBundle(locale, catalog, bundle).catch(() => undefined);
    }
  }
}

/**
 * whenCatalogsReady resolves once every required bundle has loaded for the active locale and
 * any locale being switched to. Route loading waits on it so a page renders translated on
 * its first frame rather than in English and then again.
 */
export async function whenCatalogsReady(): Promise<void> {
  await Promise.all(loadingLocales().map((locale) => settle(locale)));
}

/**
 * afterCatalogs passes a value through once the catalogs are ready. Chained onto a dynamic
 * import (`import("./panel").then(afterCatalogs)`), the module's code has evaluated and
 * required its bundles before this runs, so whatever renders it — a lazy route, a
 * React.lazy component behind its Suspense skeleton — shows the page translated on its
 * first frame instead of in English and then again. A bundle that fails to download does
 * not fail the import: the component renders in English and the bundle is retried later.
 */
export async function afterCatalogs<T>(value: T): Promise<T> {
  await whenCatalogsReady().catch(() => undefined);
  return value;
}

/**
 * registerCatalogSource adds a package's own catalog (an edition's strings) on top of the
 * app catalog. Its entries win over the app's for the same key, so a package can carry the
 * strings that exist only in its code. Catalogs already loaded are dropped and the active
 * locale reloads, so registering after startup still takes effect.
 */
export async function registerCatalogSource(source: CatalogSource): Promise<void> {
  if (!Object.values(source).some((loader) => loader !== undefined)) return;

  extraSources.push(source);
  catalogs.clear();
  if (activeLocale !== DEFAULT_LOCALE) {
    await setLocale(activeLocale);
  }
}

/**
 * loadCatalog loads everything required so far for a locale without switching to it. While
 * it runs, bundles required by code that loads in the meantime are fetched for that locale
 * too, so the switch that follows lands with nothing missing.
 */
export async function loadCatalog(locale: Locale): Promise<Messages> {
  // English needs no catalog: the source string is the key, so lookups fall through to it.
  if (locale === DEFAULT_LOCALE) return {};

  switching.set(locale, (switching.get(locale) ?? 0) + 1);
  try {
    return await settle(locale);
  } finally {
    const remaining = (switching.get(locale) ?? 1) - 1;
    if (remaining === 0) switching.delete(locale);
    else switching.set(locale, remaining);
  }
}

export async function setLocale(locale: Locale): Promise<void> {
  if (!isLocale(locale)) return;

  const messages = await loadCatalog(locale);
  if (locale !== activeLocale) activeTranslator = createTranslator(locale);
  activeLocale = locale;
  activeMessages = messages;

  if (typeof document !== "undefined") {
    document.documentElement.lang = locale;
  }

  changed();
}

function createTranslator(locale: Locale): TranslateFn {
  return (message, ...args) => translateIn(locale, message, ...args);
}

/**
 * translate looks a message up by its English source text. A missing entry falls back to
 * that source rather than rendering a key, so an untranslated string is merely English —
 * never `shipment.header.title` in front of a customer.
 */
export function translate(message: string | null | undefined, ...args: unknown[]): string {
  return translateIn(activeLocale, message, ...args);
}

/**
 * translateIn renders in an explicitly named locale. useT() binds this to the locale React
 * has rendered with, so a language switch mid-render cannot produce a component whose text
 * is half one language and half the other.
 */
export function translateIn(
  locale: Locale,
  message: string | null | undefined,
  ...args: unknown[]
): string {
  // An optional caption is ordinary — `label?: string` on a menu entry, a description a
  // row may not have. Rendering nothing matches what the bare value did before it was
  // wrapped, and is far better than making every call site guard.
  if (message === null || message === undefined || message === "") return "";

  const messages =
    locale === activeLocale ? activeMessages : (catalogs.get(locale)?.messages ?? {});
  const translated = messages[message] ?? message;
  return formatMessage(locale, translated, args);
}

export function hasTranslation(message: string): boolean {
  return message in activeMessages;
}
