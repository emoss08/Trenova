import { translate } from "@trenova/shared/i18n/runtime";
import { isNavGroup, type NavGroup, type NavItem, type NavModule } from "@/config/navigation.types";
import type { SidebarLink } from "@/components/sidebar-nav";
import type { GlobalSearchEntityType } from "@/services/global-search";
import { Settings01Icon } from "@trenova/shared/components/icons";

type PaletteIconComponent = React.ComponentType<{
  className?: string;
  size?: number;
  strokeWidth?: number;
}>;

/**
 * The live figures a page's preview can show. Each is read from a query the
 * page itself already depends on, and a figure the person cannot read, or
 * that has not loaded, is simply not drawn.
 */
export type PageGlanceKey =
  | "shipments"
  | "service-failures"
  | "workers"
  | "billing-queue"
  | "pending-approvals"
  | "reconciliation-exceptions"
  | "receivables"
  | "edi"
  | "desk";

const PAGE_GLANCES: Readonly<Record<string, PageGlanceKey>> = {
  "/shipment-management/shipments": "shipments",
  "/shipment-management/service-failures": "service-failures",
  "/hr/workers": "workers",
  "/billing/queue": "billing-queue",
  "/billing/pending-approvals": "pending-approvals",
  "/billing/reconciliation-exceptions": "reconciliation-exceptions",
  "/accounting/ar/customer-ledger": "receivables",
  "/accounting/ar/aging": "receivables",
  "/edi/overview": "edi",
  "/desk": "desk",
};

/** The pages that list one kind of record, so a preview can offer the ones opened last. */
const PAGE_RECORD_TYPES: Readonly<Record<string, GlobalSearchEntityType>> = {
  "/shipment-management/shipments": "shipment",
  "/billing/configuration-files/customers": "customer",
  "/hr/workers": "worker",
};

export interface RouteCommandItem {
  id: string;
  title: string;
  /** Where the page sits, without the page itself. Empty for a top-level page. */
  trail: string[];
  href: string;
  group: string;
  icon: PaletteIconComponent;
  keywords: string[];
  glance?: PageGlanceKey;
  recordType?: GlobalSearchEntityType;
}

export interface RouteCommandGroup {
  id: string;
  label: string;
  items: RouteCommandItem[];
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
  return normalized || "/";
}

interface CommandItemInput {
  id: string;
  title: string;
  trail: string[];
  path: string;
  group: string;
  icon: PaletteIconComponent;
  keywords: readonly string[] | undefined;
}

function createCommandItem(input: CommandItemInput): RouteCommandItem | null {
  const href = normalizePath(input.path);
  if (!href) {
    return null;
  }

  return {
    id: input.id,
    title: input.title,
    trail: input.trail,
    href,
    group: input.group,
    icon: input.icon,
    keywords: [...(input.keywords ?? [])],
    glance: PAGE_GLANCES[href],
    recordType: PAGE_RECORD_TYPES[href],
  };
}

function collectNavItems(
  commands: RouteCommandItem[],
  module: NavModule,
  items: (NavItem | NavGroup)[],
  parentSegments: string[] = [],
): void {
  for (const item of items) {
    if (isNavGroup(item)) {
      collectNavItems(commands, module, item.items, [...parentSegments, item.label]);
      continue;
    }

    if (item.disabled) {
      continue;
    }

    const command = createCommandItem({
      id: `${module.id}:${item.id}`,
      title: item.label,
      trail: [module.label, ...parentSegments],
      path: item.path,
      group: module.label,
      icon: item.icon ?? module.icon,
      keywords: item.keywords,
    });
    if (command) {
      commands.push(command);
    }
  }
}

export function buildRouteCommandGroups(
  modules: NavModule[],
  adminLinks: SidebarLink[],
): RouteCommandGroup[] {
  const groupedCommands = new Map<string, RouteCommandItem[]>();
  const addToGroup = (groupLabel: string, items: RouteCommandItem[]) => {
    if (!groupedCommands.has(groupLabel)) {
      groupedCommands.set(groupLabel, []);
    }

    groupedCommands.get(groupLabel)!.push(...items);
  };

  for (const module of modules) {
    const moduleCommands: RouteCommandItem[] = [];
    const hasSecondaryNavigation = !module.hideSecondarySidebar && module.navigation.length > 0;

    if (hasSecondaryNavigation) {
      collectNavItems(moduleCommands, module, module.navigation);
    } else {
      const command = createCommandItem({
        id: `${module.id}:base`,
        title: module.label,
        trail: [],
        path: module.basePath,
        group: module.label,
        icon: module.icon,
        keywords: module.keywords,
      });
      if (command) {
        moduleCommands.push(command);
      }
    }

    const dedupedItems = dedupeCommands(moduleCommands);
    if (dedupedItems.length === 0) {
      continue;
    }

    addToGroup(module.label, dedupedItems);
  }

  const administration = translate("Administration");
  for (const link of adminLinks) {
    if (link.disabled) {
      continue;
    }

    const group = link.group || "Administration";
    const command = createCommandItem({
      id: `admin:${link.title.toLowerCase().replace(/\s+/g, "-")}`,
      title: link.title,
      trail: link.group ? [administration, link.group] : [administration],
      path: link.href,
      group,
      icon: Settings01Icon,
      keywords: link.keywords,
    });

    if (command) {
      addToGroup(group, [command]);
    }
  }

  return Array.from(groupedCommands.entries()).map(([label, items]) => {
    const dedupedItems = dedupeCommands(items);
    return {
      id: label.toLowerCase().replace(/\s+/g, "-"),
      label,
      items: dedupedItems,
    };
  });
}

function dedupeCommands(items: RouteCommandItem[]): RouteCommandItem[] {
  const seenPaths = new Set<string>();
  const deduped: RouteCommandItem[] = [];

  for (const item of items) {
    if (seenPaths.has(item.href)) {
      continue;
    }

    seenPaths.add(item.href);
    deduped.push(item);
  }

  return deduped;
}

export function buildCommandHref(path: string, query?: Record<string, string>): string {
  const normalizedPath = normalizePath(path) ?? path;
  if (!query || Object.keys(query).length === 0) {
    return normalizedPath;
  }

  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    params.set(key, value);
  }

  return `${normalizedPath}?${params.toString()}`;
}
