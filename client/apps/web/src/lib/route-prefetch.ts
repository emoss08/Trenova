import { queryClient } from "@/lib/query-client";
import type { QueryFunctionContext, QueryKey } from "@tanstack/react-query";

// Long enough to cover the gap between the loader firing a request and the page's own
// useQuery subscribing to it, short enough that a page which opts into a shorter window
// still sees live data on its next visit. Only applies to prefetchQuery's own decision to
// fetch: an observer keeps whatever staleTime it declares.
export const PREFETCH_STALE_TIME_MS = 5_000;

// staleTime and retry may be functions of the query in TanStack's own option
// types; they are only forwarded to it, so the seam does not narrow them.
interface RoutePrefetchOptions {
  readonly queryKey: QueryKey;
  readonly staleTime?: unknown;
  readonly gcTime?: number;
  readonly retry?: unknown;
}

/**
 * One query a route wants in flight before its page mounts. The objects produced by the
 * `queries.<domain>.<entry>()` factory already have this shape, so pages hand those over
 * directly. `queryFn` is declared as a method so the tuple-keyed functions the factory
 * generates remain assignable.
 */
export interface RoutePrefetchQuery extends RoutePrefetchOptions {
  queryFn(context: QueryFunctionContext): unknown;
}

/**
 * A paged query a route wants warmed. An infinite query stores `{ pages,
 * pageParams }` where a plain one stores the payload itself, so warming one
 * through prefetchQuery would seed a cache entry the page then cannot read —
 * the page would throw on `data.pages`. Carrying the paging fields is what
 * lets `warm` reach for the right prefetch. `queryFn` is optional because
 * `infiniteQueryOptions` types it so, even when the factory always sets one.
 */
export interface RoutePrefetchInfiniteQuery extends RoutePrefetchOptions {
  queryFn?(context: QueryFunctionContext): unknown;
  readonly initialPageParam: unknown;
  readonly getNextPageParam: (...args: never[]) => unknown;
}

export type RoutePrefetchEntry = RoutePrefetchQuery | RoutePrefetchInfiniteQuery;

export type RoutePrefetchList = readonly RoutePrefetchEntry[];

/** What a page's prefetch is told about the navigation: the path's params and the query. */
export interface RoutePrefetchArgs {
  readonly params: Readonly<Record<string, string | undefined>>;
  readonly searchParams: URLSearchParams;
}

/**
 * Returns the queries a route wants warmed for a given navigation. Runs after the route's
 * gates, so it may read the auth and permission stores, the params and the query to
 * reproduce exactly the keys the page will ask for. Anything it cannot derive from them
 * should be left to the page.
 */
export type RoutePrefetch = (
  args: RoutePrefetchArgs,
) => RoutePrefetchList | Promise<RoutePrefetchList>;

/** The part of the router's loader context a prefetch reads. */
interface PrefetchLoaderContext {
  readonly params: Readonly<Record<string, string | undefined>>;
  readonly location: { readonly searchStr: string };
  readonly cause: "preload" | "enter" | "stay";
}

function isInfinite(options: RoutePrefetchEntry): options is RoutePrefetchInfiniteQuery {
  return "initialPageParam" in options;
}

function warm(options: RoutePrefetchEntry): void {
  const shared = { ...options, staleTime: options.staleTime ?? PREFETCH_STALE_TIME_MS };

  // The page type is opaque at this seam: every entry carries its own queryFn,
  // and the two agree by construction inside the factory that produced them.
  const started = isInfinite(options)
    ? queryClient.prefetchInfiniteQuery(
        shared as unknown as Parameters<typeof queryClient.prefetchInfiniteQuery>[0],
      )
    : queryClient.prefetchQuery(
        shared as unknown as Parameters<typeof queryClient.prefetchQuery>[0],
      );

  void started.catch(() => undefined);
}

async function runPrefetch(prefetch: RoutePrefetch, args: RoutePrefetchArgs): Promise<void> {
  const list = await prefetch(args);
  for (const options of list) {
    warm(options);
  }
}

/**
 * Wraps a prefetch list as a route loader that never blocks navigation: the fetches are
 * started and the loader returns immediately. A failed prefetch is not a failed navigation —
 * the page's own query will surface the error where the user can see it. A navigation that
 * stays on the page (a change of the query alone) warms nothing: the page's queries are
 * already subscribed and follow the query themselves.
 */
export function createPrefetchLoader(prefetch: RoutePrefetch) {
  return ({ params, location, cause }: PrefetchLoaderContext): void => {
    if (cause === "stay") {
      return;
    }
    const searchParams = new URLSearchParams(location.searchStr);
    runPrefetch(prefetch, { params, searchParams }).catch(() => undefined);
  };
}

function exportsPrefetch(module: object): module is { prefetch: RoutePrefetch } {
  return "prefetch" in module && typeof module.prefetch === "function";
}

/**
 * Lets a page colocate its prefetch list with its component. The importer is the same
 * dynamic import the route's lazy component uses, so both resolve the one module promise the
 * bundler already holds and the page chunk is never loaded twice. A page module that
 * exports nothing called `prefetch` simply warms nothing.
 */
export function lazyPrefetch(importer: () => Promise<object>): RoutePrefetch {
  return async (args) => {
    const module = await importer();
    return exportsPrefetch(module) ? module.prefetch(args) : [];
  };
}

/** The loader of a route whose page module exports `prefetch`. */
export function prefetchPage(importer: () => Promise<object>) {
  return createPrefetchLoader(lazyPrefetch(importer));
}
