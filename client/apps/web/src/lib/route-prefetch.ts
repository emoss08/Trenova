import type { QueryClient, QueryFunctionContext, QueryKey } from "@tanstack/react-query";

// Long enough to cover the gap between the loader (or a predicted link) firing a request and
// the page's own useQuery subscribing to it, short enough that a page which opts into a
// shorter window still sees live data on its next visit. prefetchQuery skips a fetch inside
// it, and the query client's refetchOnMount keeps data this young when the page mounts
// (`lib/query-client.ts`); an observer otherwise keeps whatever staleTime it declares.
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

/** The part of the router's loader context a page's loader reads. */
interface PageLoaderContext {
  readonly params: Readonly<Record<string, string | undefined>>;
  readonly location: { readonly searchStr: string };
  readonly cause: "preload" | "enter" | "stay";
  readonly context: { readonly queryClient: QueryClient };
}

function isInfinite(options: RoutePrefetchEntry): options is RoutePrefetchInfiniteQuery {
  return "initialPageParam" in options;
}

function warm(queryClient: QueryClient, options: RoutePrefetchEntry): void {
  const shared = { ...options, staleTime: options.staleTime ?? PREFETCH_STALE_TIME_MS };

  // The page type is opaque at this seam: every entry carries its own queryFn,
  // and the two agree by construction inside the factory that produced them.
  const started = isInfinite(options)
    ? queryClient.prefetchInfiniteQuery(
        shared as unknown as Parameters<QueryClient["prefetchInfiniteQuery"]>[0],
      )
    : queryClient.prefetchQuery(shared as unknown as Parameters<QueryClient["prefetchQuery"]>[0]);

  void started.catch(() => undefined);
}

function isOnScreen(queryClient: QueryClient, options: RoutePrefetchEntry): boolean {
  const query = queryClient.getQueryCache().find({ queryKey: options.queryKey, exact: true });
  return (query?.getObserversCount() ?? 0) > 0;
}

async function runPrefetch(
  queryClient: QueryClient,
  prefetch: RoutePrefetch,
  args: RoutePrefetchArgs,
  cause: PageLoaderContext["cause"],
): Promise<void> {
  const list = await prefetch(args);
  for (const options of list) {
    // A preload never touches a query something on screen is reading: that component keeps it
    // live and refetches it when it should. Warming it anyway refetched the whole page a person
    // was on whenever the pointer crossed a link to it, and redrew every reader of the result.
    if (cause === "preload" && isOnScreen(queryClient, options)) continue;
    warm(queryClient, options);
  }
}

/**
 * The loader of a page that knows what it will read. The page's chunk (its component and this
 * loader, split together) has loaded and waited for its strings by the time this runs, so it
 * starts the prefetch and returns without waiting for it: a failed prefetch is not a failed
 * navigation, and the page's own query surfaces the error where the person can see it. A
 * navigation that stays on the page (a change of the query alone) warms nothing: the page's
 * queries are subscribed and follow the query themselves.
 */
export function pageLoader(prefetch: RoutePrefetch) {
  return ({ params, location, cause, context }: PageLoaderContext): void => {
    if (cause === "stay") {
      return;
    }
    const searchParams = new URLSearchParams(location.searchStr);
    runPrefetch(context.queryClient, prefetch, { params, searchParams }, cause).catch(
      () => undefined,
    );
  };
}
