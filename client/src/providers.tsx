import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { NuqsAdapter } from "nuqs/adapters/react-router/v7";
import type React from "react";
import { RootErrorBoundary } from "./components/error-boundary";
import { ThemeProvider } from "./components/theme-provider";
import { Toaster } from "./components/ui/toaster";
// import { ApiRequestError } from "./lib/api";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // retry: (failureCount, error) => {
      //   if (error instanceof ApiRequestError && error.isAuthenticationError()) {
      //     return false;
      //   }
      //   return failureCount < 3;
      // },
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
});

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <RootErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <NuqsAdapter>
          <ThemeProvider defaultTheme="system" storageKey="trenova-ui-theme">
            {children}
            {/*<ReactQueryDevtools
              buttonPosition="bottom-left"
              initialIsOpen={false}
            />*/}
            <Toaster position="top-center" />
          </ThemeProvider>
        </NuqsAdapter>
      </QueryClientProvider>
    </RootErrorBoundary>
  );
}
