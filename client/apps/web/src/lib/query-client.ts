import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { reportFailure } from "@trenova/shared/lib/failure-reporter";
import { retryDelay, shouldRetry } from "@/lib/query-retry";

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
      staleTime: 0,
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
});
