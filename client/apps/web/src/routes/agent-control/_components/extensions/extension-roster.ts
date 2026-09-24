import type { AgentExtensionCatalogItem } from "@/types/agent-extension";

export const extensionStatusFilters = ["all", "on", "off"] as const;
export type ExtensionStatusFilter = (typeof extensionStatusFilters)[number];

export const extensionSorts = ["featured", "name", "newest"] as const;
export type ExtensionSort = (typeof extensionSorts)[number];

export const ALL_CATEGORIES = "all";

export type ExtensionState = "on" | "needsSetup" | "off";

export type ExtensionFilter = {
  query: string;
  category: string;
  status: ExtensionStatusFilter;
};

/**
 * Where an extension stands for the organization. "Needs setup" is an
 * extension switched on without the credentials it runs on, which agents
 * cannot use, so it reads as neither on nor off.
 */
export function extensionState(item: AgentExtensionCatalogItem): ExtensionState {
  if (item.enabled && item.configured) {
    return "on";
  }
  if (item.enabled) {
    return "needsSetup";
  }
  return "off";
}

function matchesQuery(item: AgentExtensionCatalogItem, query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return true;
  }

  const haystack = [
    item.name,
    item.vendor,
    item.summary,
    item.categoryLabel,
    ...item.capabilities,
    ...item.tools.map((tool) => `${tool.name} ${tool.label}`),
  ]
    .join(" ")
    .toLowerCase();

  return needle.split(/\s+/).every((word) => haystack.includes(word));
}

export function filterExtensions(
  items: AgentExtensionCatalogItem[],
  filter: ExtensionFilter,
): AgentExtensionCatalogItem[] {
  return items.filter((item) => {
    if (filter.category !== ALL_CATEGORIES && item.category !== filter.category) {
      return false;
    }
    if (filter.status === "on" && extensionState(item) !== "on") {
      return false;
    }
    if (filter.status === "off" && extensionState(item) === "on") {
      return false;
    }
    return matchesQuery(item, filter.query);
  });
}

export function sortExtensions(
  items: AgentExtensionCatalogItem[],
  sort: ExtensionSort,
): AgentExtensionCatalogItem[] {
  const byName = (a: AgentExtensionCatalogItem, b: AgentExtensionCatalogItem) =>
    a.name.localeCompare(b.name) || a.vendor.localeCompare(b.vendor);

  return [...items].sort((a, b) => {
    switch (sort) {
      case "name":
        return byName(a, b);
      case "newest":
        return b.releasedAt - a.releasedAt || byName(a, b);
      default:
        if (a.featured !== b.featured) {
          return a.featured ? -1 : 1;
        }
        return a.sortOrder - b.sortOrder || byName(a, b);
    }
  });
}

/** How much of the day's request limit the organization has used, from 0 to 1. */
export function dailyUsageShare(item: AgentExtensionCatalogItem): number {
  if (item.dailyRequestLimit <= 0) {
    return 0;
  }
  return Math.min(1, item.usage.requestsToday / item.dailyRequestLimit);
}
