import { appAdminLinks, appNavigationModules } from "@/config/app-navigation";
import { recordPath } from "@/config/record-links";
import { isNavGroup } from "@/config/navigation.types";
import { normalizePath } from "@/lib/navigation-paths";
import { navigationSections } from "@/lib/navigation-sections";

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

function createRouteTitleIndex(): RouteTitleEntry[] {
  const collectedEntries: RouteTitleEntry[] = navigationSections.map(({ path, label }) => ({
    path,
    label,
  }));

  for (const module of appNavigationModules) {
    for (const item of module.navigation.flatMap((entry) =>
      isNavGroup(entry) ? entry.items : [entry],
    )) {
      const normalizedPath = normalizePath(item.path);
      if (normalizedPath) {
        collectedEntries.push({ path: normalizedPath, label: item.label });
      }
    }
  }

  for (const link of appAdminLinks) {
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

/**
 * Where a crumb's label came from: a page that named the record it shows, the
 * navigation (a page or a section, either of which a person can open), or the
 * URL segment itself, which names a folder nothing fills.
 */
export type BreadcrumbSource = "published" | "navigation" | "derived";

export interface BreadcrumbSegment {
  path: string;
  label: string;
  source: BreadcrumbSource;
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
 * The source says which, so a trail can leave out folders that lead nowhere.
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
    const published = labelOverrides[path];
    const navigation = routeTitleByPath.get(path);
    if (published !== undefined) {
      crumbs.push({ path, label: published, source: "published" });
    } else if (navigation !== undefined) {
      crumbs.push({ path, label: navigation, source: "navigation" });
    } else {
      crumbs.push({ path, label: formatSegmentLabel(segment), source: "derived" });
    }
    index = end;
  }

  return crumbs;
}

/**
 * The pages above this one that a person can open, in the words they read
 * them in. Every crumb is a link that lands somewhere: a page, a section
 * index, or a record a page has named. A folder of the URL that nothing fills
 * ("/accounting/journal-entries") is left out rather than drawn as a link to
 * a missing page, and a crumb that only repeats the title is dropped, so the
 * Shipments board reads "Shipments" once rather than three times.
 */
export function trailCrumbs(
  breadcrumbs: readonly BreadcrumbSegment[],
  title: string,
  translate: (label: string) => string,
): { path: string; label: string }[] {
  const last = breadcrumbs.at(-1);
  const lastLabel = last && (last.source === "navigation" ? translate(last.label) : last.label);

  return breadcrumbs
    .slice(0, -1)
    .filter((crumb) => crumb.source !== "derived")
    .map((crumb) => ({
      path: crumb.path,
      label: crumb.source === "navigation" ? translate(crumb.label) : crumb.label,
    }))
    .filter((crumb) => crumb.label !== title && crumb.label !== lastLabel);
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
  return recordPath("worker", workerId, tab ? { tab } : undefined);
}
