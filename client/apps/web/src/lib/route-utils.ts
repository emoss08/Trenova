import { adminLinks, navigationConfig } from "@/config/navigation.config";
import type { NavGroup, NavItem } from "@/config/navigation.types";

interface RouteTitleEntry {
  path: string;
  label: string;
}

const PULID_PATTERN = /^[a-z]{2,10}_[0-9A-Za-z]{20,30}$/;

function formatSegmentLabel(segment: string): string {
  if (PULID_PATTERN.test(segment)) {
    return "Details";
  }

  return segment
    .replace(/[-_]/g, " ")
    .replace(/([A-Z])/g, " $1")
    .replace(/\w\S*/g, (txt) => txt.charAt(0).toUpperCase() + txt.substring(1).toLowerCase())
    .trim();
}

function stripTrailingSlash(path: string): string {
  return path.endsWith("/") ? path.slice(0, -1) : path;
}

export function isRouteActive(currentPath: string, itemPath?: string): boolean {
  if (!itemPath) {
    return false;
  }
  if (itemPath === "/") {
    return currentPath === "/";
  }

  const current = stripTrailingSlash(currentPath);
  const target = stripTrailingSlash(itemPath);
  return current === target || current.startsWith(`${target}/`);
}

/**
 * Resolves the single best-matching nav path for the current location using
 * longest-prefix-wins, so a parent path (e.g. `/reports`) does not stay active
 * when a more specific sibling (e.g. `/reports/runs`) matches.
 */
export function findActiveNavPath(
  currentPath: string,
  candidatePaths: readonly string[],
): string | null {
  let bestMatch: string | null = null;
  let bestLength = -1;

  for (const candidate of candidatePaths) {
    if (!isRouteActive(currentPath, candidate)) {
      continue;
    }
    const length = stripTrailingSlash(candidate).length;
    if (length > bestLength) {
      bestMatch = candidate;
      bestLength = length;
    }
  }

  return bestMatch;
}

export function normalizePath(path: string): string | null {
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

/**
 * The directory every path in the list shares, or null when they share
 * nothing beyond the root.
 */
function sharedDirectory(paths: readonly string[]): string | null {
  if (paths.length === 0) {
    return null;
  }
  let common = paths[0].split("/").slice(0, -1);
  for (const path of paths.slice(1)) {
    const parts = path.split("/").slice(0, -1);
    let length = 0;
    while (length < common.length && length < parts.length && common[length] === parts[length]) {
      length += 1;
    }
    common = common.slice(0, length);
  }
  const directory = common.join("/");
  return directory.length > 1 ? directory : null;
}

/**
 * Pages label their own paths. A group labels the directory its pages share,
 * so "/accounting/ar" reads as the group is named rather than as "Ar", but
 * only when that directory belongs to the group alone: a group whose pages
 * sit directly under the module has nothing of its own to name.
 */
function collectNavItemEntries(
  entries: RouteTitleEntry[],
  items: (NavItem | NavGroup)[],
  moduleRoots: ReadonlySet<string>,
): void {
  for (const item of items) {
    if ("items" in item) {
      const before = entries.length;
      collectNavItemEntries(entries, item.items, moduleRoots);
      const groupPaths = entries.slice(before).map((entry) => entry.path);
      const directory = sharedDirectory(groupPaths);
      if (directory && !moduleRoots.has(directory) && !groupPaths.includes(directory)) {
        entries.push({ path: directory, label: item.label });
      }
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
    // A module owns its base path and every prefix it declares; those are
    // its roots, and no group underneath may claim them. The base path
    // carries the full name; a declared prefix carries the short one, since
    // the base path is often a page inside it whose crumb follows.
    const moduleRoots = new Set<string>();
    const moduleBasePath = normalizePath(module.basePath);
    if (moduleBasePath) {
      moduleRoots.add(moduleBasePath);
      collectedEntries.push({ path: moduleBasePath, label: module.label });
    }
    for (const prefix of module.routePrefixes ?? []) {
      const normalizedPrefix = normalizePath(prefix);
      if (normalizedPrefix && !moduleRoots.has(normalizedPrefix)) {
        moduleRoots.add(normalizedPrefix);
        collectedEntries.push({
          path: normalizedPrefix,
          label: module.shortLabel ?? module.label,
        });
      }
    }

    collectNavItemEntries(collectedEntries, module.navigation, moduleRoots);
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
const routeTitleByPath = new Map(routeTitleIndex.map((entry) => [entry.path, entry.label]));

export interface BreadcrumbSegment {
  path: string;
  label: string;
}

/**
 * Action verbs that describe what the page does to the record identified by
 * the segment before them. They fold into the record's crumb rather than
 * standing as one of their own, so "/things/<id>/edit" yields a single crumb
 * for the record instead of "Details / Edit".
 */
const RECORD_ACTION_SEGMENTS: ReadonlySet<string> = new Set(["edit", "view"]);

/**
 * Splits a pathname into cumulative crumbs. Each crumb takes its label from,
 * in order: a label a page published for that exact path, the navigation
 * config's label for that exact path, or a Title Case rendering of the segment.
 */
export function generateBreadcrumbSegments(
  pathname: string,
  labelOverrides: Readonly<Record<string, string>> = {},
): BreadcrumbSegment[] {
  const parts = pathname.replace(/\/$/, "").split("/").filter(Boolean);
  const crumbs: BreadcrumbSegment[] = [];

  let index = 0;
  while (index < parts.length) {
    const segment = parts[index];
    let end = index + 1;
    if (
      PULID_PATTERN.test(segment) &&
      end < parts.length &&
      RECORD_ACTION_SEGMENTS.has(parts[end])
    ) {
      end += 1;
    }

    const path = "/" + parts.slice(0, end).join("/");
    const label = labelOverrides[path] ?? routeTitleByPath.get(path) ?? formatSegmentLabel(segment);
    crumbs.push({ path, label });
    index = end;
  }

  return crumbs;
}

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
  const pathSegments = pathname.replace(/^\//, "").replace(/\/$/, "").split("/").filter(Boolean);

  if (pathSegments.length === 0) return "Home";

  // Take the last segment as the page title
  const lastSegment = pathSegments[pathSegments.length - 1];
  return lastSegment.replace(/-/g, " ").replace(/\b\w/g, (l) => l.toUpperCase());
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

/**
 * The link that opens a worker's record from anywhere in the app. The page
 * reads the entity from the query string, so a tab can be named as well.
 */
export function workerRecordHref(workerId: string, tab?: string): string {
  const base = `/hr/workers?entityId=${encodeURIComponent(workerId)}&modType=edit`;
  return tab ? `${base}&tab=${encodeURIComponent(tab)}` : base;
}
