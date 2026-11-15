import { MutationCache, QueryCache, QueryClient, type Query } from "@tanstack/react-query";
import { reportFailure } from "@trenova/shared/lib/failure-reporter";
import { retryDelay, shouldRetry } from "@/lib/query-retry";
import { PREFETCH_STALE_TIME_MS } from "@/lib/route-prefetch";

/**
 * Whether mounting a query fetches it again. Queries default to stale at once, so without
 * this a page fetched its data a second time the moment it mounted on data a route's loader
 * or a predicted link had just warmed. Data fetched within `PREFETCH_STALE_TIME_MS` is what
 * the page came for and is kept; anything older, and anything invalidated (a save, a realtime
 * event), is fetched as before. Only consulted for a stale query: a query that declares a
 * longer staleTime keeps it.
 */
export function refetchOnMount(query: Query): boolean {
  return (
    query.state.isInvalidated || Date.now() - query.state.dataUpdatedAt >= PREFETCH_STALE_TIME_MS
  );
}

// Module-level so route loaders can warm the cache before a lazily loaded page mounts.
// providers.tsx hands this same instance to the QueryClientProvider, which is what lets a
// prefetch started in a loader be picked up by the page's own useQuery.
export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) =>
      reportFailure(error, { source: "query", operation: String(query.queryKey[0]) }),
  }),
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) =>
      reportFailure(error, {
        source: "mutation",
        operation: String(mutation.options.mutationKey?.[0] ?? "mutation"),
      }),
  }),
  defaultOptions: {
    queries: {
      retry: shouldRetry,
      retryDelay,
      refetchOnWindowFocus: false,
      refetchOnMount,
      staleTime: 0,
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
});
