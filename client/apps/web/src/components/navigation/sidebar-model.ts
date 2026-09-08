import type { SidebarLink } from "@/components/sidebar-nav";
import { ATTENTION_ROWS_BY_KEY, type AttentionTone } from "@/config/attention-rows";
import { appModuleGroups } from "@/config/navigation.config";
import { isNavGroup, type ModuleId, type NavItem, type NavModule } from "@/config/navigation.types";
import { isRouteActive } from "@/lib/route-utils";

export interface SidebarPageItem {
  id: string;
  label: string;
  path: string;
  disabled?: boolean;
  includeBetaTag?: boolean;
}

export interface SidebarPageSection {
  id: string;
  label: string | null;
  items: SidebarPageItem[];
}

/**
 * One module as the sidebar draws it: the pages people work in, grouped the
 * way the module groups them, with the configuration catalogue set apart so
 * it can be rendered as a footer rather than a peer.
 */
export interface SidebarModuleView {
  module: NavModule;
  sections: SidebarPageSection[];
  configuration: SidebarPageItem[];
  landingPath: string;
}

export interface SidebarDomain {
  id: string;
  label: string;
  modules: NavModule[];
}

export interface ModuleAttention {
  count: number;
  tone: AttentionTone;
}

const OTHER_DOMAIN_ID = "other";
const OTHER_DOMAIN_LABEL = "More";
const UNGROUPED_ADMIN_LABEL = "Other";

const TONE_RANK: Record<AttentionTone, number> = {
  default: 0,
  warning: 1,
  destructive: 2,
};

export function moduleDisplayLabel(module: NavModule): string {
  return module.shortLabel ?? module.label;
}

function moduleRoutePrefixes(module: NavModule): readonly string[] {
  if (module.routePrefixes && module.routePrefixes.length > 0) {
    return module.routePrefixes;
  }
  return module.basePath === "#" ? [] : [module.basePath];
}

/**
 * The module a location belongs to, longest prefix wins. Home only claims the
 * root so it never shadows a module whose prefix happens to start with `/`.
 */
export function findModuleForPath(
  modules: readonly NavModule[],
  pathname: string,
): NavModule | null {
  let best: NavModule | null = null;
  let bestLength = -1;

  for (const module of modules) {
    if (module.basePath === "/") {
      if (pathname === "/" && bestLength < 1) {
        best = module;
        bestLength = 1;
      }
      continue;
    }

    for (const prefix of moduleRoutePrefixes(module)) {
      if (prefix.length > bestLength && isRouteActive(pathname, prefix)) {
        best = module;
        bestLength = prefix.length;
      }
    }
  }

  return best;
}

function pageItemFromNavItem(item: NavItem): SidebarPageItem {
  return {
    id: item.id,
    label: item.label,
    path: item.path,
    disabled: item.disabled,
    includeBetaTag: item.includeBetaTag,
  };
}

function pageItemFromAdminLink(link: SidebarLink, group: string): SidebarPageItem {
  return {
    id: `${group}:${link.href}`,
    label: link.title,
    path: link.href,
    disabled: link.disabled,
    includeBetaTag: link.includeBetaTag,
  };
}

function adminSections(links: readonly SidebarLink[]): SidebarPageSection[] {
  const sections = new Map<string, SidebarPageSection>();
  for (const link of links) {
    const group = link.group ?? UNGROUPED_ADMIN_LABEL;
    const item = pageItemFromAdminLink(link, group);
    const existing = sections.get(group);
    if (existing) {
      existing.items.push(item);
    } else {
      sections.set(group, { id: group, label: group, items: [item] });
    }
  }
  return Array.from(sections.values());
}

function moduleSections(module: NavModule): {
  sections: SidebarPageSection[];
  configuration: SidebarPageItem[];
} {
  const sections: SidebarPageSection[] = [];
  const configuration: SidebarPageItem[] = [];
  let pending: SidebarPageSection | null = null;

  for (const entry of module.navigation) {
    if (!isNavGroup(entry)) {
      pending ??= { id: `${module.id}:${entry.id}`, label: null, items: [] };
      pending.items.push(pageItemFromNavItem(entry));
      continue;
    }

    if (entry.kind === "configuration") {
      for (const item of entry.items) {
        configuration.push(pageItemFromNavItem(item));
      }
      continue;
    }

    if (pending) {
      sections.push(pending);
      pending = null;
    }
    sections.push({
      id: entry.id,
      label: entry.label,
      items: entry.items.map(pageItemFromNavItem),
    });
  }

  if (pending) {
    sections.push(pending);
  }

  return { sections, configuration };
}

function landingPathOf(
  module: NavModule,
  sections: readonly SidebarPageSection[],
  configuration: readonly SidebarPageItem[],
): string {
  for (const section of sections) {
    const first = section.items.find((item) => !item.disabled);
    if (first) {
      return first.path;
    }
  }
  const firstConfig = configuration.find((item) => !item.disabled);
  return firstConfig?.path ?? module.basePath;
}

export function buildModuleView(
  module: NavModule,
  adminLinks: readonly SidebarLink[],
): SidebarModuleView {
  if (module.id === "admin") {
    return {
      module,
      sections: adminSections(adminLinks),
      configuration: [],
      landingPath: module.basePath,
    };
  }

  const { sections, configuration } = moduleSections(module);
  return {
    module,
    sections,
    configuration,
    landingPath: landingPathOf(module, sections, configuration),
  };
}

/**
 * Modules in their configured domains. A module no domain claims still has
 * to be reachable, so it lands in a trailing group instead of vanishing.
 */
export function groupModulesByDomain(modules: readonly NavModule[]): SidebarDomain[] {
  const byId = new Map<ModuleId, NavModule>(modules.map((module) => [module.id, module]));
  const claimed = new Set<ModuleId>();
  const domains: SidebarDomain[] = [];

  for (const group of appModuleGroups) {
    const members: NavModule[] = [];
    for (const id of group.moduleIds) {
      const module = byId.get(id);
      if (module) {
        members.push(module);
        claimed.add(id);
      }
    }
    if (members.length > 0) {
      domains.push({ id: group.id, label: group.label, modules: members });
    }
  }

  const stray = modules.filter((module) => !claimed.has(module.id));
  if (stray.length > 0) {
    domains.push({ id: OTHER_DOMAIN_ID, label: OTHER_DOMAIN_LABEL, modules: stray });
  }

  return domains;
}

export function collectNavPaths(views: readonly SidebarModuleView[]): string[] {
  const paths = new Set<string>();
  for (const view of views) {
    for (const section of view.sections) {
      for (const item of section.items) {
        paths.add(item.path);
      }
    }
    for (const item of view.configuration) {
      paths.add(item.path);
    }
  }
  return Array.from(paths);
}

/**
 * The watched counts keyed by the page they belong to, so the row for that
 * page can carry its number.
 */
export function pageAttention(
  summary: Readonly<Record<string, number | null | undefined>> | null | undefined,
  metricKeys: readonly string[],
): Map<string, ModuleAttention> {
  const result = new Map<string, ModuleAttention>();
  if (!summary) {
    return result;
  }

  for (const key of metricKeys) {
    const row = ATTENTION_ROWS_BY_KEY.get(key);
    const count = summary[key];
    if (!row || !count || count <= 0) {
      continue;
    }
    result.set(row.path, { count, tone: row.tone });
  }

  return result;
}

/**
 * Attention counts folded onto the module that owns each destination, so a
 * module row can carry one number instead of the sidebar carrying a section.
 */
export function moduleAttention(
  modules: readonly NavModule[],
  summary: Readonly<Record<string, number | null | undefined>> | null | undefined,
  metricKeys: readonly string[],
): Map<ModuleId, ModuleAttention> {
  const result = new Map<ModuleId, ModuleAttention>();
  if (!summary) {
    return result;
  }

  for (const key of metricKeys) {
    const row = ATTENTION_ROWS_BY_KEY.get(key);
    const count = summary[key];
    if (!row || !count || count <= 0) {
      continue;
    }

    const module = findModuleForPath(modules, row.path);
    if (!module) {
      continue;
    }

    const existing = result.get(module.id);
    if (!existing) {
      result.set(module.id, { count, tone: row.tone });
      continue;
    }

    existing.count += count;
    if (TONE_RANK[row.tone] > TONE_RANK[existing.tone]) {
      existing.tone = row.tone;
    }
  }

  return result;
}
