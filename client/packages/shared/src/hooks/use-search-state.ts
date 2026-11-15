import {
  replaceEqualDeep,
  useLocation,
  useRouter,
  useRouterState,
  type AnyRouter,
} from "@tanstack/react-router";
import type { Nullable, Options, UseQueryStatesKeysMap, Values } from "nuqs";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useCallback, useState } from "react";

// The router owns the address: these hooks read the state a page keeps there from the router's
// location and write it back through one navigation. nuqs's parsers are the codecs (how a value
// is written and read back, its default); nothing else of nuqs runs.

/** How a write lands: a new history entry or the same one, and whether the page scrolls up. */
export type SearchWriteOptions = Pick<Options, "history" | "scroll">;

type SearchMap = UseQueryStatesKeysMap;

/** The values a parser map reads from the address. */
export type SearchState<M extends SearchMap> = Values<M>;
type Patch<M extends SearchMap> = Partial<Nullable<Values<M>>>;

/** What a write changes: some keys (null removes one), or a function of what the address holds. */
export type SearchStateUpdate<M extends SearchMap> =
  | Patch<M>
  | null
  | ((current: Values<M>) => Patch<M> | null);

export type SetSearchState<M extends SearchMap> = (
  update: SearchStateUpdate<M>,
  options?: SearchWriteOptions,
) => Promise<void>;

type AnyParser = SearchMap[string] & {
  history?: "push" | "replace";
  scroll?: boolean;
  clearOnDefault?: boolean;
  defaultValue?: unknown;
  eq?: (a: unknown, b: unknown) => boolean;
  serialize: (value: never) => string;
  parseServerSide?: (value: string | string[] | undefined) => unknown;
  parse: (value: string) => unknown;
};

function rawValue(value: unknown): string | undefined {
  if (typeof value === "string") return value;
  if (Array.isArray(value)) return typeof value[0] === "string" ? value[0] : undefined;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return undefined;
}

function parseKey(parser: AnyParser, raw: string | undefined): unknown {
  if (parser.parseServerSide) {
    return parser.parseServerSide(raw);
  }
  const parsed = raw === undefined ? null : parser.parse(raw);
  return parsed ?? parser.defaultValue ?? null;
}

/**
 * The values `map` names, read from a query as the router holds it (or a URLSearchParams, as a
 * route's prefetch is handed): a key that is missing or does not parse reads as its default, or
 * null without one.
 */
export function readSearchState<M extends SearchMap>(
  map: M,
  search: Readonly<Record<string, unknown>> | URLSearchParams,
): Values<M> {
  const record = search instanceof URLSearchParams ? Object.fromEntries(search) : search;
  const values: Record<string, unknown> = {};
  for (const [key, parser] of Object.entries(map)) {
    values[key] = parseKey(parser as AnyParser, rawValue(record[key]));
  }
  return values as Values<M>;
}

function sameValue(parser: AnyParser, a: unknown, b: unknown): boolean {
  if (a === null || b === null || a === undefined || b === undefined) return a === b;
  return parser.eq ? parser.eq(a, b) : a === b;
}

/**
 * Writes `patch` over `current` (both as the router holds the query). A value removed, equal to
 * its key's default, or written as nothing leaves the key out of the address.
 */
function writeSearchState(
  map: SearchMap,
  current: Readonly<Record<string, unknown>>,
  patch: Readonly<Record<string, unknown>>,
): Record<string, unknown> {
  const next = new Map(Object.entries(current));
  for (const [key, value] of Object.entries(patch)) {
    const parser = map[key] as AnyParser | undefined;
    if (value === null || value === undefined || !parser) {
      next.delete(key);
      continue;
    }
    const clearsDefault = parser.clearOnDefault ?? true;
    if (
      clearsDefault &&
      parser.defaultValue !== undefined &&
      sameValue(parser, value, parser.defaultValue)
    ) {
      next.delete(key);
      continue;
    }
    const serialized = parser.serialize(value as never);
    if (serialized === "") {
      next.delete(key);
    } else {
      next.set(key, serialized);
    }
  }
  return Object.fromEntries(next);
}

function historyOf(map: SearchMap, keys: string[], options: SearchWriteOptions | undefined) {
  if (options?.history) return options.history;
  return keys.some((key) => (map[key] as AnyParser | undefined)?.history === "push")
    ? "push"
    : "replace";
}

function scrollOf(map: SearchMap, keys: string[], options: SearchWriteOptions | undefined) {
  if (options?.scroll !== undefined) return options.scroll;
  return keys.some((key) => (map[key] as AnyParser | undefined)?.scroll === true);
}

/** Navigates to the current path with `search` as its query, keeping its hash and its state. */
function commitSearch(
  router: AnyRouter,
  search: Record<string, unknown>,
  { replace, resetScroll }: { replace: boolean; resetScroll: boolean },
): Promise<void> {
  const { pathname, hash } = router.latestLocation;
  const query = router.options.stringifySearch(search);
  return router.navigate({
    href: `${pathname}${query}${hash ? `#${hash}` : ""}`,
    replace,
    resetScroll,
    state: (current) => current,
  });
}

interface PendingSearch {
  search: Record<string, unknown>;
  push: boolean;
  resetScroll: boolean;
  committed: Promise<void>;
}

const pendingSearches = new WeakMap<AnyRouter, PendingSearch>();

/** What one change makes of the query, and how it asks the address to be written. */
export interface SearchChange {
  search: Record<string, unknown>;
  /** A new history entry rather than the same one. */
  push: boolean;
  /** Back to the top of the page. */
  resetScroll: boolean;
}

/**
 * Changes the address's query. Every change made in the same moment, from any hook, is applied
 * on top of the ones before it and the address is written once: one navigation, one history
 * entry (a new one when any change asked to push), so a hook that changes two kinds of state
 * at once (a row expanded and its editor opened) never loses the first to the second.
 */
export function updateSearch(
  router: AnyRouter,
  change: (current: Readonly<Record<string, unknown>>) => SearchChange,
): Promise<void> {
  let pending = pendingSearches.get(router);
  if (!pending) {
    let settle!: { resolve: () => void; reject: (error: unknown) => void };
    const committed = new Promise<void>((resolve, reject) => {
      settle = { resolve, reject };
    });
    const created: PendingSearch = {
      search: { ...(router.latestLocation.search as Record<string, unknown>) },
      push: false,
      resetScroll: false,
      committed,
    };
    pending = created;
    pendingSearches.set(router, created);
    queueMicrotask(() => {
      pendingSearches.delete(router);
      commitSearch(router, created.search, {
        replace: !created.push,
        resetScroll: created.resetScroll,
      }).then(settle.resolve, settle.reject);
    });
  }
  const next = change(pending.search);
  pending.search = next.search;
  pending.push ||= next.push;
  pending.resetScroll ||= next.resetScroll;
  return pending.committed;
}

/**
 * The state a page keeps in the address, under the keys `map` names, and the way to change it.
 *
 * Values are read from the router's location as it changes, so a field bound to one follows
 * every keystroke. While a navigation is on its way to another page, the page being left keeps
 * reading the address it was drawn at, so nothing on it reacts to the next page's query. A
 * change is one navigation built from the address as it stands when it runs: two changes in
 * the same moment keep each other, and nothing else writes the address in between. It is a
 * new history entry when the call, the hook or one of the keys it changes asks for "push".
 */
export function useSearchState<M extends SearchMap>(
  map: M,
  options?: SearchWriteOptions,
): [Values<M>, SetSearchState<M>] {
  const router = useRouter();
  const search = useLocation({
    select: (location) => location.search as Record<string, unknown>,
  });
  const pathname = useLocation({ select: (location) => location.pathname });
  const settledPathname = useRouterState({
    select: (state) => state.resolvedLocation?.pathname ?? state.location.pathname,
  });

  // What this page was last drawn with: the address it was at and the values it read there. A
  // value that reads the same as it did stays the same object (the router's structural
  // sharing), so an effect keyed on a list of filters does not run again when only the page
  // changes. While a navigation takes the address elsewhere, the page keeps what it drew.
  const [drawn, setDrawn] = useState(() => ({ pathname, values: readSearchState(map, search) }));
  const leaving = pathname !== settledPathname && drawn.pathname !== pathname;
  const read = leaving
    ? drawn.values
    : (replaceEqualDeep(drawn.values, readSearchState(map, search)) as Values<M>);
  if (!leaving && (drawn.pathname !== pathname || drawn.values !== read)) {
    setDrawn({ pathname, values: read });
  }

  const setSearchState: SetSearchState<M> = useLatestCallback((update, callOptions) => {
    const merged = { ...options, ...callOptions };
    return updateSearch(router, (current) => {
      const patch = typeof update === "function" ? update(readSearchState(map, current)) : update;
      const cleared =
        patch === null ? Object.fromEntries(Object.keys(map).map((key) => [key, null])) : patch;
      const keys = Object.keys(cleared);
      return {
        search: writeSearchState(map, current, cleared),
        push: historyOf(map, keys, merged) === "push",
        resetScroll: scrollOf(map, keys, merged),
      };
    });
  });

  return [read, setSearchState];
}

export type SetSearchValue<T> = (
  value: T | null | ((current: T) => T | null),
  options?: SearchWriteOptions,
) => Promise<void>;

/** What useSearchValue reads a key with: a nuqs parser, with or without a default. */
interface SearchParser<T> {
  parse: (value: string) => T | null;
  serialize: (value: T) => string;
}

/** One key of the state a page keeps in the address; see useSearchState. */
export function useSearchValue<T>(
  key: string,
  parser: SearchParser<T> & { readonly defaultValue: NonNullable<T> },
  options?: SearchWriteOptions,
): [NonNullable<T>, SetSearchValue<NonNullable<T>>];
export function useSearchValue<T>(
  key: string,
  parser: SearchParser<T>,
  options?: SearchWriteOptions,
): [T | null, SetSearchValue<T | null>];
export function useSearchValue<T>(
  key: string,
  parser: SearchParser<T>,
  options?: SearchWriteOptions,
): [T | null, SetSearchValue<T | null>] {
  const [values, setValues] = useSearchState(
    { [key]: parser } as unknown as UseQueryStatesKeysMap,
    options,
  );
  const setValue = useCallback<SetSearchValue<T | null>>(
    (value, callOptions) =>
      setValues(
        (current) => ({
          [key]:
            typeof value === "function"
              ? (value as (current: T | null) => T | null)(current[key] as T | null)
              : value,
        }),
        callOptions,
      ),
    [key, setValues],
  );
  return [values[key] as T | null, setValue];
}
