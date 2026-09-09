import type { OrganizationCapabilityType } from "@trenova/shared/types/organization-capability";
import type { OperationType } from "@trenova/shared/types/permission";
import type { LucideIcon } from "lucide-react";

export type ModuleId =
  | "home"
  | "dispatch"
  | "hr"
  | "fleet"
  | "fuel"
  | "billing"
  | "detention"
  | "payroll"
  | "carrier-settlements"
  | "edi"
  | "reports"
  | "accounting"
  | "organization"
  | "admin"
  | "shipment";

export interface NavBadge {
  content: string | number;
  variant: "default" | "destructive" | "warning";
}

export type NavItemBadgeKind = "edi-attention";

export interface NavItem {
  id: string;
  label: string;
  path: string;
  icon?: LucideIcon;
  disabled?: boolean;
  includeBetaTag?: boolean;
  external?: boolean;
  resource?: string;
  /**
   * Hides the entry when the organization has the capability turned off. It sits
   * alongside `resource` rather than replacing it: permissions are access
   * control, a capability is only whether the organization uses that half of the
   * product at all.
   */
  capability?: OrganizationCapabilityType;
  badge?: NavItemBadgeKind;
}

export type NavGroupKind = "configuration";

export interface NavGroup {
  id: string;
  label: string;
  icon?: LucideIcon;
  items: NavItem[];
  defaultOpen?: boolean;
  /**
   * Marks a group the sidebar demotes below a module's working pages. The
   * catalogues a module is configured with are reached far less often than
   * the records it manages, so they render as a quiet footer instead of a
   * peer of the module's pages.
   */
  kind?: NavGroupKind;
  resource?: string;
  capability?: OrganizationCapabilityType;
}

export interface NavModule {
  id: ModuleId;
  label: string;
  /**
   * The name the sidebar shows when space is scarce. Breadcrumbs, page titles
   * and the command palette keep using `label`.
   */
  shortLabel?: string;
  icon: React.ComponentType<{
    className?: string;
    size?: number;
    strokeWidth?: number;
  }>;
  description?: string;
  basePath: string;
  /**
   * Route prefixes that belong to the module when `basePath` alone cannot
   * say so: a module whose pages live under a different prefix than its
   * landing route, or one that owns several prefixes.
   */
  routePrefixes?: readonly string[];
  navigation: (NavItem | NavGroup)[];
  hideSecondarySidebar?: boolean;
  resource?: string;
  capability?: OrganizationCapabilityType;
}

export interface NavigationConfig {
  modules: NavModule[];
  quickActions?: QuickActionCommand[];
}

export interface QuickActionCommand {
  id: string;
  label: string;
  description: string;
  path: string;
  resource?: string;
  requiredOperation?: OperationType;
  /**
   * Mirrors `NavItem.capability`: a create shortcut for a half of the product
   * the organization does not run would only lead to a route that refuses it.
   */
  capability?: OrganizationCapabilityType;
  query?: Record<string, string>;
  keywords?: string[];
}

export function isNavGroup(item: NavItem | NavGroup): item is NavGroup {
  return "items" in item;
}

export function getFirstNavPath(module: NavModule): string {
  for (const item of module.navigation) {
    if (isNavGroup(item)) {
      if (item.items.length > 0) {
        return item.items[0].path;
      }
    } else {
      return item.path;
    }
  }
  return module.basePath;
}
