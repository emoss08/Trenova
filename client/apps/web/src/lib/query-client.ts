import { QueryClient } from "@tanstack/react-query";
import { retryDelay, shouldRetry } from "@/lib/query-retry";

// Module-level so route loaders can warm the cache before a lazily loaded page mounts.
// providers.tsx hands this same instance to the QueryClientProvider, which is what lets a
// prefetch started in a loader be picked up by the page's own useQuery.
export const queryClient = new QueryClient({
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
