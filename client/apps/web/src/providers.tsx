import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { NuqsAdapter } from "nuqs/adapters/react-router/v7";
import type React from "react";
import { RootErrorBoundary } from "@trenova/shared/components/error-boundary";
import { ThemeProvider } from "@trenova/shared/components/theme-provider";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import { setPartialErrorReporter, setSessionExpiryHandler } from "@trenova/shared/lib/graphql";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { retryDelay, shouldRetry } from "@/lib/query-retry";

const queryClient = new QueryClient({
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

// The transport cannot reach the router or the auth store, so the expiry path is wired
// here instead. clearAuth performs the same teardown protectedLoader's 401 branch does;
// the reload is what a loader's redirect() would have done had one been running.
setSessionExpiryHandler(() => {
  const { isAuthenticated } = useAuthStore.getState();
  if (!isAuthenticated) {
    return;
  }

  useAuthStore.getState().clearAuth();

  if (window.location.pathname !== "/login") {
    window.location.assign("/login");
  }
});

setPartialErrorReporter((errors, { operationName }) => {
  for (const error of errors) {
    console.warn(
      `[graphql] ${operationName ?? "operation"} returned partial data: ${error.message}`,
      { code: error.code, path: error.path, traceId: error.traceId },
    );
  }
});

export function normalizeSearchParams(search: URLSearchParams) {
  const nextSearch = new URLSearchParams(search);

  removeEmptyDataTableParams(nextSearch);
  normalizeOrganizationSettingsSearchParams(nextSearch);

  return nextSearch;
}

function removeEmptyDataTableParams(search: URLSearchParams) {
  for (const key of ["fieldFilters", "filterGroups", "sort"]) {
    const value = search.get(key);
    if (value === "" || value === "[]") {
      search.delete(key);
    }
  }

  if (search.get("pageIndex") === "1") {
    search.delete("pageIndex");
  }
}

function normalizeOrganizationSettingsSearchParams(search: URLSearchParams) {
  if (typeof window === "undefined") {
    return;
  }

  const pathname = window.location.pathname.replace(/\/+$/, "");
  if (pathname !== "/admin/organization-settings") {
    return;
  }

  const tab = search.get("tab") || "general";
  if (tab !== "security") {
    for (const key of [
      "securityTab",
      "activityView",
      "directoryId",
      "search",
      "editingProvider",
      "panelMode",
      "panelOpen",
    ]) {
      search.delete(key);
    }
    return;
  }

  const securityTab = search.get("securityTab") || "sign-in";
  if (securityTab !== "provisioning") {
    search.delete("directoryId");
  }
  if (securityTab !== "activity") {
    search.delete("activityView");
  }
  if (securityTab !== "sign-in") {
    for (const key of ["search", "editingProvider", "panelMode", "panelOpen"]) {
      search.delete(key);
    }
  }
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <RootErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <NuqsAdapter processUrlSearchParams={normalizeSearchParams}>
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
