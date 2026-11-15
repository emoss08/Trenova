import { NuqsAdapter } from "nuqs/adapters/tanstack-router";
import type { ReactNode } from "react";

/**
 * Where nuqs reads and writes the query string: through the router, so a state change is a
 * navigation the router knows about. It sits inside the router (its `InnerWrap`), and every
 * write passes through `normalizeSearchParams` first.
 */
export function UrlStateAdapter({ children }: { children: ReactNode }) {
  return <NuqsAdapter processUrlSearchParams={normalizeSearchParams}>{children}</NuqsAdapter>;
}

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
