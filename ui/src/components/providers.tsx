import { APIError } from "@/types/errors";
import { createSyncStoragePersister } from "@tanstack/query-sync-storage-persister";
import { QueryClient } from "@tanstack/react-query";
// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { PersistQueryClientProvider } from "@tanstack/react-query-persist-client";
import { compress, decompress } from "lz-string";
import { NuqsAdapter } from "nuqs/adapters/react-router";
import { HelmetProvider } from "react-helmet-async";
import { ThemeProvider } from "./theme-provider";
import { Toaster } from "./ui/sonner";

const persister = createSyncStoragePersister({
  storage: window.localStorage,
  serialize: (data) => compress(JSON.stringify(data)),
  deserialize: (data) => JSON.parse(decompress(data)),
});

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
      <PersistQueryClientProvider
        persistOptions={{
          persister,
          maxAge: Infinity,
        }}
        client={queryClient}
      >
        <NuqsAdapter>
          <ThemeProvider defaultTheme="dark" storageKey="trenova-ui-theme">
            {/* <ReactQueryDevtools /> */}
            {children}
            <Toaster />
          </ThemeProvider>
        </NuqsAdapter>
      </PersistQueryClientProvider>
    </HelmetProvider>
  );
}
