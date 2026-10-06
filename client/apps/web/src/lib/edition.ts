import editionEntry from "@trenova/edition-entry";
import { resolveEdition, type EditionAdminLink, type EditionNavItem } from "@trenova/edition";
import type { SidebarLink } from "@/components/sidebar-nav";
import type { NavGroup, NavItem, NavModule } from "@/config/navigation.types";
import { isNavGroup } from "@/config/navigation.types";

/**
 * The edition this build was compiled with: the overlay in client/packages/cloud when it
 * is present, the self-hosted no-op otherwise. Everything the web app renders or routes
 * for an edition goes through this one object. See docs/engineering/editions.md.
 */
export const edition = resolveEdition(editionEntry);

function insertAfter<T>(list: T[], value: T, matches: (entry: T) => boolean): void {
  const index = list.findIndex(matches);
  if (index === -1) {
    list.push(value);
    return;
  }
  list.splice(index + 1, 0, value);
}

function toSidebarLink({ after: _after, ...link }: EditionAdminLink): SidebarLink {
  return link;
}

/** The host's admin links with an edition's inserted where each one asks to go. */
export function mergeAdminLinks(
  base: readonly SidebarLink[],
  additions: readonly EditionAdminLink[],
): SidebarLink[] {
  if (additions.length === 0) {
    return [...base];
  }

  const merged = [...base];
  for (const addition of additions) {
    const link = toSidebarLink(addition);
    if (addition.after === undefined) {
      merged.push(link);
      continue;
    }
    insertAfter(merged, link, (entry) => entry.href === addition.after);
  }
  return merged;
}

function toNavItem(item: EditionNavItem["item"]): NavItem {
  return item;
}

function placeItem(
  entries: (NavItem | NavGroup)[],
  addition: EditionNavItem,
): (NavItem | NavGroup)[] {
  const item = toNavItem(addition.item);
  const next = [...entries];

  if (addition.groupId !== undefined) {
    const groupIndex = next.findIndex(
      (entry) => isNavGroup(entry) && entry.id === addition.groupId,
    );
    const group = next[groupIndex];
    if (groupIndex !== -1 && isNavGroup(group)) {
      const items = [...group.items];
      if (addition.after === undefined) {
        items.push(item);
      } else {
        insertAfter(items, item, (entry) => entry.id === addition.after);
      }
      next[groupIndex] = { ...group, items };
      return next;
    }
  }

  if (addition.after === undefined) {
    next.push(item);
  } else {
    insertAfter(next, item, (entry) => entry.id === addition.after);
  }
  return next;
}

/**
 * The host's navigation modules with an edition's entries placed into them. An entry
 * naming a module the host does not have is dropped: there is nowhere to show it.
 */
export function mergeNavItems(
  modules: readonly NavModule[],
  additions: readonly EditionNavItem[],
): NavModule[] {
  if (additions.length === 0) {
    return [...modules];
  }

  return modules.map((module) => {
    const mine = additions.filter((addition) => addition.moduleId === module.id);
    if (mine.length === 0) {
      return module;
    }
    return { ...module, navigation: mine.reduce(placeItem, module.navigation) };
  });
}
