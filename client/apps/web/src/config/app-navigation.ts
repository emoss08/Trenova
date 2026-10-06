import type { SidebarLink } from "@/components/sidebar-nav";
import { adminLinks, navigationConfig } from "@/config/navigation.config";
import type { NavModule } from "@/config/navigation.types";
import { edition, mergeAdminLinks, mergeNavItems } from "@/lib/edition";

/**
 * The navigation the app renders: the host's own config with the edition's entries
 * placed into it. navigation.config.ts stays the host's literal, which the product
 * guide generator reads statically.
 */
export const appNavigationModules: readonly NavModule[] = mergeNavItems(
  navigationConfig.modules,
  edition.navItems,
);

export const appAdminLinks: readonly SidebarLink[] = mergeAdminLinks(
  adminLinks,
  edition.adminLinks,
);
