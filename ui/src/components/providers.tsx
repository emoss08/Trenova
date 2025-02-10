import { APIError } from "@/types/errors";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { NuqsAdapter } from "nuqs/adapters/react-router";
import { HelmetProvider } from "react-helmet-async";
import { ThemeProvider } from "./theme-provider";
import { Toaster } from "./ui/sonner";

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
    <HelmetProvider>
      <QueryClientProvider client={queryClient}>
        <NuqsAdapter>
          <ThemeProvider defaultTheme="dark" storageKey="trenova-ui-theme">
            {/* <ReactQueryDevtools /> */}
            {children}
            <Toaster />
          </ThemeProvider>
        </NuqsAdapter>
      </QueryClientProvider>
    </HelmetProvider>
  );
}
