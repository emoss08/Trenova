import { navigationConfig } from "@/config/navigation.config";
import {
  isNavGroup,
  type NavGroup,
  type NavItem,
  type NavModule,
} from "@/config/navigation.types";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";
import { useMemo } from "react";

export function useFilteredNavigation() {
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);

  return useMemo(() => {
    if (!manifest) {
      return navigationConfig.modules;
    }

    const canAccessItem = (item: NavItem | NavGroup | NavModule): boolean => {
      if (item.resource) {
        return hasPermission(item.resource, Operation.Read);
      }

      return true;
    };

    const filterNavItems = (
      items: (NavItem | NavGroup)[],
    ): (NavItem | NavGroup)[] => {
      return items
        .map((item) => {
          if (isNavGroup(item)) {
            const filteredItems = item.items.filter(canAccessItem);
            if (filteredItems.length === 0) {
              return null;
            }
            return { ...item, items: filteredItems };
          }
          return canAccessItem(item) ? item : null;
        })
        .filter((item): item is NavItem | NavGroup => item !== null);
    };

    const filteredModules = navigationConfig.modules
      .filter(canAccessItem)
      .map((module) => ({
        ...module,
        navigation: filterNavItems(module.navigation),
      }))
      .filter((module) => {
        if (module.hideSecondarySidebar) return true;
        return module.navigation.length > 0;
      });

    return filteredModules;
  }, [manifest, hasPermission]);
}

export function useFilteredModuleNavigation(module: NavModule | null) {
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);

  return useMemo(() => {
    if (!module || !manifest) {
      return module?.navigation ?? [];
    }

    const canAccessItem = (item: NavItem | NavGroup): boolean => {
      if (item.resource) {
        return hasPermission(item.resource, Operation.Read);
      }

      return true;
    };

    return module.navigation
      .map((item) => {
        if (isNavGroup(item)) {
          const filteredItems = item.items.filter(canAccessItem);
          if (filteredItems.length === 0) {
            return null;
          }
          return { ...item, items: filteredItems };
        }
        return canAccessItem(item) ? item : null;
      })
      .filter((item): item is NavItem | NavGroup => item !== null);
  }, [module, manifest, hasPermission]);
}
