import type { SidebarLink } from "@/components/sidebar-nav";
import { appAdminLinks } from "@/config/app-navigation";
import { normalizePath } from "@/lib/navigation-paths";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { isFeatureOnPlan, useUnavailableFeatures } from "@trenova/shared/hooks/use-plan-feature";
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
  const unavailableFeatures = useUnavailableFeatures();
  const { config } = usePublicConfig();
  const platformMode = config.platformMode;
  const managed = config.managed;

  return useMemo(
    () =>
      appAdminLinks.filter((link) => {
        if (link.disabled) {
          return false;
        }

        if (link.capability && !hasOrganizationCapability(capabilities, link.capability)) {
          return false;
        }

        if (!isFeatureOnPlan(unavailableFeatures, link.planFeature)) {
          return false;
        }

        if (link.platformMode && link.platformMode !== platformMode) {
          return false;
        }

        if (link.managedOnly && !managed) {
          return false;
        }

        if (!manifest) {
          return false;
        }

        if (link.anyResources) {
          return link.anyResources.some((resource) => hasPermission(resource, Operation.Read));
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
    [
      canAccessRoute,
      capabilities,
      hasPermission,
      manifest,
      unavailableFeatures,
      platformMode,
      managed,
    ],
  );
}
