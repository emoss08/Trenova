import type { OperationType } from "@/types/permission";
import type { LucideIcon } from "lucide-react";

export type ModuleId =
  | "home"
  | "dispatch"
  | "fleet"
  | "billing"
  | "reports"
  | "accounting"
  | "organization"
  | "admin"
  | "shipment";

export interface NavBadge {
  content: string | number;
  variant: "default" | "destructive" | "warning";
}

export interface NavItem {
  id: string;
  label: string;
  path: string;
  icon?: LucideIcon;
  disabled?: boolean;
  includeBetaTag?: boolean;
  external?: boolean;
  resource?: string;
  adminOnly?: boolean;
}

export interface NavGroup {
  id: string;
  label: string;
  icon?: LucideIcon;
  items: NavItem[];
  defaultOpen?: boolean;
  resource?: string;
  adminOnly?: boolean;
}

export interface NavModule {
  id: ModuleId;
  label: string;
  icon: React.ComponentType<{
    className?: string;
    size?: number;
    strokeWidth?: number;
  }>;
  description?: string;
  basePath: string;
  navigation: (NavItem | NavGroup)[];
  hideSecondarySidebar?: boolean;
  resource?: string;
  adminOnly?: boolean;
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
