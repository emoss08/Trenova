/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { routes as navRoutes } from "@/lib/nav-links";
import { RouteInfo } from "@/types/nav-links";

// utils/route-utils.ts
export function generateBreadcrumbSegments(pathname: string) {
  // Remove trailing slash and split into segments
  const segments = pathname.replace(/\/$/, "").split("/").filter(Boolean);

  // Generate cumulative paths and readable labels
  return segments.map((segment, index) => {
    const path = "/" + segments.slice(0, index + 1).join("/");
    // Convert kebab-case or camelCase to Title Case
    const label = segment
      .replace(/[-_]/g, " ")
      .replace(/([A-Z])/g, " $1")
      .replace(
        /\w\S*/g,
        (txt) => txt.charAt(0).toUpperCase() + txt.substring(1).toLowerCase(),
      )
      .trim();

    return { path, label };
  });
}

// Helper function to check if a route is active (same logic as app-sidebar)
const isRouteActive = (currentPath: string, itemPath?: string): boolean => {
  if (!itemPath) return false;

  // Special handling for root path
  if (itemPath === "/") {
    return currentPath === "/";
  }

  // For other paths, ensure we match complete segments
  const normalizedCurrentPath = currentPath.endsWith("/")
    ? currentPath.slice(0, -1)
    : currentPath;
  const normalizedItemPath = itemPath.endsWith("/")
    ? itemPath.slice(0, -1)
    : itemPath;

  return normalizedCurrentPath.startsWith(normalizedItemPath);
};

// Helper function to find matching route in nav hierarchy
const findMatchingRoute = (
  currentPath: string,
  routes: RouteInfo[],
): RouteInfo | null => {
  for (const route of routes) {
    // Check if this route matches
    if (route.link && isRouteActive(currentPath, route.link)) {
      // If it has children, check for a more specific match
      if (route.tree) {
        const childMatch = findMatchingRoute(currentPath, route.tree);
        if (childMatch) {
          return childMatch;
        }
      }
      return route;
    }

    // Check children even if parent doesn't match
    if (route.tree) {
      const childMatch = findMatchingRoute(currentPath, route.tree);
      if (childMatch) {
        return childMatch;
      }
    }
  }

  return null;
};

/**
 * Extracts the page title from nav-links configuration based on the current pathname
 */
export function getPageTitleFromRoute(pathname: string): string | null {
  const matchingRoute = findMatchingRoute(pathname, navRoutes);
  return matchingRoute?.label || null;
}

/**
 * Generates a fallback title from pathname when no route title is found
 */
export function generateFallbackTitle(pathname: string): string {
  const pathSegments = pathname
    .replace(/^\//, "")
    .replace(/\/$/, "")
    .split("/")
    .filter(Boolean);

  if (pathSegments.length === 0) return "Home";

  // Take the last segment as the page title
  const lastSegment = pathSegments[pathSegments.length - 1];
  return lastSegment
    .replace(/-/g, " ")
    .replace(/\b\w/g, (l) => l.toUpperCase());
}

/**
 * Gets the page title, first trying to match from routes, then falling back to pathname parsing
 */
export function getPageTitle(pathname: string): string {
  const routeTitle = getPageTitleFromRoute(pathname);

  if (routeTitle) {
    return routeTitle;
  }

  return generateFallbackTitle(pathname);
}
