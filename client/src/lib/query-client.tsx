import { QueryClient } from "@tanstack/react-query";

export const tanstackQueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 1000 * 60 * 60 * 2, // 1 hour
      gcTime: 1000 * 60 * 60 * 24, // 24 hours
    },
  },
});
