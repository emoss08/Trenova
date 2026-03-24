import { adminLinks, navigationConfig } from "@/config/navigation.config";
import type { NavGroup, NavItem } from "@/config/navigation.types";

interface RouteTitleEntry {
  path: string;
  label: string;
}

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

function normalizePath(path: string): string | null {
  if (!path || path === "#") {
    return null;
  }

  const trimmed = path.trim();
  if (!trimmed || trimmed === "#") {
    return null;
  }

  const withLeadingSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  const normalized = withLeadingSlash.replace(/\/+$/, "");

  return normalized === "" ? "/" : normalized;
}

function collectNavItemEntries(
  entries: RouteTitleEntry[],
  items: (NavItem | NavGroup)[],
): void {
  for (const item of items) {
    if ("items" in item) {
      collectNavItemEntries(entries, item.items);
      continue;
    }

    const normalizedPath = normalizePath(item.path);
    if (normalizedPath) {
      entries.push({ path: normalizedPath, label: item.label });
    }
  }
}

function createRouteTitleIndex(): RouteTitleEntry[] {
  const collectedEntries: RouteTitleEntry[] = [];

  for (const module of navigationConfig.modules) {
    const moduleBasePath = normalizePath(module.basePath);
    if (moduleBasePath) {
      collectedEntries.push({ path: moduleBasePath, label: module.label });
    }

    collectNavItemEntries(collectedEntries, module.navigation);
  }

  for (const link of adminLinks) {
    const normalizedPath = normalizePath(link.href);
    if (normalizedPath) {
      collectedEntries.push({ path: normalizedPath, label: link.title });
    }
  }

  const dedupedEntries = new Map<string, RouteTitleEntry>();
  for (const entry of collectedEntries) {
    if (!dedupedEntries.has(entry.path)) {
      dedupedEntries.set(entry.path, entry);
    }
  }

  return Array.from(dedupedEntries.values());
}

function isPathMatch(pathname: string, candidatePath: string): boolean {
  if (candidatePath === "/") {
    return pathname === "/";
  }

  return pathname === candidatePath || pathname.startsWith(`${candidatePath}/`);
}

const routeTitleIndex = createRouteTitleIndex();

function findMatchingRoute(pathname: string): RouteTitleEntry | null {
  const normalizedPath = normalizePath(pathname);
  if (!normalizedPath) {
    return null;
  }

  let bestMatch: RouteTitleEntry | null = null;

  for (const entry of routeTitleIndex) {
    if (!isPathMatch(normalizedPath, entry.path)) {
      continue;
    }

    if (!bestMatch || entry.path.length > bestMatch.path.length) {
      bestMatch = entry;
    }
  }

  return bestMatch;
}

/**
 * Extracts the page title from navigation configuration based on the current pathname
 */
export function getPageTitleFromRoute(pathname: string): string | null {
  const matchingRoute = findMatchingRoute(pathname);
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
