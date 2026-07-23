import { isNavGroup, type NavGroup, type NavItem, type NavModule } from "@/config/navigation.types";
import type { SidebarLink } from "@/components/sidebar-nav";
import type { QuickActionCommand } from "@/config/navigation.types";
import { Operation, type OperationType } from "@/types/permission";
import { SettingsIcon } from "lucide-react";

type PaletteIconComponent = React.ComponentType<{
  className?: string;
  size?: number;
  strokeWidth?: number;
}>;

export interface RouteCommandItem {
  id: string;
  title: string;
  subtitle: string;
  href: string;
  group: string;
  icon: PaletteIconComponent;
  keywords: string[];
}

export interface RouteCommandGroup {
  id: string;
  label: string;
  items: RouteCommandItem[];
}

export interface SuggestedCommandItem {
  id: string;
  label: string;
  description: string;
  href: string;
  icon: PaletteIconComponent;
  keywords: string[];
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

function createCommandItem(
  itemId: string,
  title: string,
  subtitle: string,
  path: string,
  group: string,
  icon: PaletteIconComponent,
  keywords: string[],
): RouteCommandItem | null {
  const normalizedPath = normalizePath(path);
  if (!normalizedPath) {
    return null;
  }

  return {
    id: itemId,
    title,
    subtitle,
    href: normalizedPath,
    group,
    icon,
    keywords,
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

    const subtitleSegments = [module.label, ...parentSegments, item.label];
    const command = createCommandItem(
      `${module.id}:${item.id}`,
      item.label,
      subtitleSegments.join(" > "),
      item.path,
      module.label,
      item.icon ?? module.icon,
      [item.label, module.label, ...parentSegments],
    );
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
      const command = createCommandItem(
        `${module.id}:base`,
        module.label,
        module.label,
        module.basePath,
        module.label,
        module.icon,
        [module.label],
      );
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

  for (const link of adminLinks) {
    if (link.disabled) {
      continue;
    }

    const group = link.group || "Administration";
    const title = link.title;
    const subtitle = link.group
      ? `Administration > ${link.group} > ${link.title}`
      : `Administration > ${link.title}`;
    const command = createCommandItem(
      `admin:${title.toLowerCase().replace(/\s+/g, "-")}`,
      title,
      subtitle,
      link.href,
      group,
      SettingsIcon,
      ["administration", group, title],
    );

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

function getRequiredOperation(action: QuickActionCommand): OperationType {
  return action.requiredOperation ?? Operation.Create;
}

export function buildSuggestedCreateCommands(
  definitions: QuickActionCommand[],
  routeGroups: RouteCommandGroup[],
  hasPermission: (resource: string, operation: OperationType) => boolean,
): SuggestedCommandItem[] {
  const routeIconByPath = new Map<string, PaletteIconComponent>();
  for (const group of routeGroups) {
    for (const item of group.items) {
      routeIconByPath.set(item.href, item.icon);
    }
  }

  return definitions
    .filter((definition) => {
      if (!definition.resource) {
        return true;
      }

      return hasPermission(definition.resource, getRequiredOperation(definition));
    })
    .map((definition) => ({
      id: definition.id,
      label: definition.label,
      description: definition.description,
      href: buildCommandHref(definition.path, definition.query),
      icon: routeIconByPath.get(buildCommandHref(definition.path)) ?? SettingsIcon,
      keywords: definition.keywords ?? [],
    }));
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
