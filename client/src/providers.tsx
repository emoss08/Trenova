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
      staleTime: 0,
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
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
