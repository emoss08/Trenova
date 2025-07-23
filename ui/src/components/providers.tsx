/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { APIError } from "@/types/errors";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { NuqsAdapter } from "nuqs/adapters/react-router";
import { ThemeProvider } from "./theme-provider";
import { Toaster } from "./ui/sonner";
import { WebSocketProvider } from "./websocket-provider";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (error instanceof APIError && error.isAuthenticationError()) {
          return false;
        }
        return failureCount < 3;
      },
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000,
      gcTime: 10 * 60 * 1000,
    },
  },
});

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <NuqsAdapter>
      <HelmetProvider>
        <QueryClientProvider client={queryClient}>
          <WebSocketProvider>
            <ThemeProvider defaultTheme="dark" storageKey="trenova-ui-theme">
              {/* <ReactQueryDevtools /> */}
              {children}
              <Toaster position="top-center" />
            </ThemeProvider>
          </WebSocketProvider>
        </QueryClientProvider>
      </HelmetProvider>
    </NuqsAdapter>
  );
}
