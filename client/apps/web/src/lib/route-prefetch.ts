import { queryClient } from "@/lib/query-client";
import type { QueryFunctionContext, QueryKey } from "@tanstack/react-query";
import type { LoaderFunction, LoaderFunctionArgs } from "react-router";

// Long enough to cover the gap between the loader firing a request and the page's own
// useQuery subscribing to it, short enough that a page which opts into a shorter window
// still sees live data on its next visit. Only applies to prefetchQuery's own decision to
// fetch: an observer keeps whatever staleTime it declares.
export const PREFETCH_STALE_TIME_MS = 5_000;

/**
 * One query a route wants in flight before its page mounts. The objects produced by the
 * `queries.<domain>.<entry>()` factory already have this shape, so pages hand those over
 * directly. `queryFn` is declared as a method so the tuple-keyed functions the factory
 * generates remain assignable.
 */
export interface RoutePrefetchQuery {
  readonly queryKey: QueryKey;
  queryFn(context: QueryFunctionContext): unknown;
  readonly staleTime?: number;
  readonly gcTime?: number;
  readonly retry?: boolean | number;
}

/**
 * A paged query a route wants warmed. An infinite query stores `{ pages,
 * pageParams }` where a plain one stores the payload itself, so warming one
 * through prefetchQuery would seed a cache entry the page then cannot read —
 * the page would throw on `data.pages`. Carrying the paging fields is what
 * lets `warm` reach for the right prefetch.
 */
export interface RoutePrefetchInfiniteQuery extends RoutePrefetchQuery {
  readonly initialPageParam: unknown;
  readonly getNextPageParam: (...args: never[]) => unknown;
}

export type RoutePrefetchEntry = RoutePrefetchQuery | RoutePrefetchInfiniteQuery;

export type RoutePrefetchList = readonly RoutePrefetchEntry[];

/**
 * Returns the queries a route wants warmed for a given navigation. Runs after the auth and
 * permission loaders, so it may read the auth and permission stores and the request URL
 * to reproduce exactly the keys the page will ask for. Anything it cannot derive from the
 * request should be left to the page.
 */
export type RoutePrefetch = (
  args: LoaderFunctionArgs,
) => RoutePrefetchList | Promise<RoutePrefetchList>;

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

async function runPrefetch(prefetch: RoutePrefetch, args: LoaderFunctionArgs): Promise<void> {
  const list = await prefetch(args);
  for (const options of list) {
    warm(options);
  }
}

/**
 * Wraps a prefetch list as a loader that never blocks navigation: the fetches are started
 * and the loader returns immediately. A failed prefetch is not a failed navigation — the
 * page's own query will surface the error where the user can see it.
 */
export function createPrefetchLoader(prefetch: RoutePrefetch): LoaderFunction {
  return (args) => {
    runPrefetch(prefetch, args).catch(() => undefined);
    return null;
  };
}

function exportsPrefetch(module: object): module is { prefetch: RoutePrefetch } {
  return "prefetch" in module && typeof module.prefetch === "function";
}

/**
 * Lets a page colocate its prefetch list with its component. The importer is the same
 * dynamic import the route's `lazy()` uses, so both resolve the one module promise the
 * bundler already holds and the page chunk is never loaded twice. A page module that
 * exports nothing called `prefetch` simply warms nothing.
 */
export function lazyPrefetch(importer: () => Promise<object>): RoutePrefetch {
  return async (args) => {
    const module = await importer();
    return exportsPrefetch(module) ? module.prefetch(args) : [];
  };
}
