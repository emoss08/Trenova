import type { SidebarLink } from "@/components/sidebar-nav";
import { adminLinks } from "@/config/navigation.config";
import { normalizePath } from "@/lib/route-utils";
import { usePermissionStore } from "@/stores/permission-store";
import { Operation } from "@/types/permission";
import { useMemo } from "react";

export function useAccessibleAdminLinks(): SidebarLink[] {
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const canAccessRoute = usePermissionStore((state) => state.canAccessRoute);

  return useMemo(
    () =>
      adminLinks.filter((link) => {
        if (link.disabled) {
          return false;
        }

        if (!manifest) {
          return false;
        }

        if (link.resource) {
          return hasPermission(link.resource, link.requiredOperation ?? Operation.Read);
        }

        const normalizedPath = normalizePath(link.href);
        if (!normalizedPath) {
          return false;
        }

        return canAccessRoute(normalizedPath) || canAccessRoute(link.href);
      }),
    [canAccessRoute, hasPermission, manifest],
  );
}
