import { appNavigationModules } from "@/config/app-navigation";
import { isNavGroup, type ModuleId, type NavModule } from "@/config/navigation.types";
import { normalizePath } from "@/lib/navigation-paths";

/**
 * A folder of the app that the navigation names but no page of its own fills:
 * a module's root ("/dispatch") or the folder a group's pages share
 * ("/accounting/ar"). The router mounts the section index on every one that
 * has no page, so every name the trail shows leads somewhere.
 */
export interface NavigationSection {
  path: string;
  label: string;
  moduleId: ModuleId;
  /** The group whose pages the section lists, or null for the whole module. */
  groupId: string | null;
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
 * A module owns its base path and every prefix it declares. The base path
 * carries the module's full name; a declared prefix the short one, since the
 * base path is often a page inside it whose crumb follows.
 */
function moduleRootSections(module: NavModule): NavigationSection[] {
  const sections: NavigationSection[] = [];
  const add = (path: string | null, label: string) => {
    if (path && path !== "/" && !sections.some((section) => section.path === path)) {
      sections.push({ path, label, moduleId: module.id, groupId: null });
    }
  };
  add(normalizePath(module.basePath), module.label);
  for (const prefix of module.routePrefixes ?? []) {
    add(normalizePath(prefix), module.shortLabel ?? module.label);
  }
  return sections;
}

/**
 * A group names the directory its pages share, so "/accounting/ar" reads as
 * the group is named rather than as "Ar", but only when that directory
 * belongs to the group alone: a group whose pages sit directly under the
 * module has nothing of its own to name.
 */
function groupSections(module: NavModule, roots: ReadonlySet<string>): NavigationSection[] {
  const sections: NavigationSection[] = [];
  for (const entry of module.navigation) {
    if (!isNavGroup(entry)) {
      continue;
    }
    const paths = entry.items
      .map((item) => normalizePath(item.path))
      .filter((path): path is string => path !== null);
    const directory = sharedDirectory(paths);
    if (directory && !roots.has(directory)) {
      sections.push({
        path: directory,
        label: entry.label,
        moduleId: module.id,
        groupId: entry.id,
      });
    }
  }
  return sections;
}

/** Every section the navigation names, the first claim on a path winning. */
export function buildNavigationSections(modules: readonly NavModule[]): NavigationSection[] {
  const byPath = new Map<string, NavigationSection>();
  for (const module of modules) {
    const roots = moduleRootSections(module);
    const rootPaths = new Set(roots.map((section) => section.path));
    for (const section of [...roots, ...groupSections(module, rootPaths)]) {
      if (!byPath.has(section.path)) {
        byPath.set(section.path, section);
      }
    }
  }
  return Array.from(byPath.values());
}

export const navigationSections: readonly NavigationSection[] =
  buildNavigationSections(appNavigationModules);

const sectionsByPath = new Map(navigationSections.map((section) => [section.path, section]));

export function findNavigationSection(pathname: string): NavigationSection | null {
  const path = normalizePath(pathname);
  return path ? (sectionsByPath.get(path) ?? null) : null;
}
