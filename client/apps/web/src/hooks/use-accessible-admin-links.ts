import type { SidebarLink } from "@/components/sidebar-nav";
import { appAdminLinks } from "@/config/app-navigation";
import { usePlanRestrictions } from "@/hooks/use-plan-restrictions";
import { isPlanRestricted } from "@/lib/plan-capability";
import { normalizePath } from "@/lib/route-utils";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { hasOrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation } from "@trenova/shared/types/permission";
import { useMemo } from "react";

export function useAccessibleAdminLinks(): SidebarLink[] {
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const canAccessRoute = usePermissionStore((state) => state.canAccessRoute);
  const capabilities = useOrgCapabilities();
  const planRestrictions = usePlanRestrictions();
  const { config } = usePublicConfig();
  const platformMode = config.platformMode;

  return useMemo(
    () =>
      appAdminLinks.filter((link) => {
        if (link.disabled) {
          return false;
        }

        if (link.capability && !hasOrganizationCapability(capabilities, link.capability)) {
          return false;
        }

        if (isPlanRestricted(planRestrictions, link.planCapability)) {
          return false;
        }

        if (link.platformMode && link.platformMode !== platformMode) {
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
    [canAccessRoute, capabilities, hasPermission, manifest, planRestrictions, platformMode],
  );
}
