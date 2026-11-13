import { buildCommandHref } from "@/components/command-palette/route-command-data";
import { navigationConfig } from "@/config/navigation.config";
import type { QuickActionCommand } from "@/config/navigation.types";
import { QUICK_ACTION_ICONS } from "@/config/quick-action-icons";
import { canAccessQuickAction } from "@/hooks/use-filtered-navigation";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { IconComponent } from "@trenova/shared/components/icons";
import { useMemo } from "react";

export interface SidebarQuickAction {
  definition: QuickActionCommand;
  icon: IconComponent;
  href: string;
}

const QUICK_ACTIONS_BY_ID = new Map(
  (navigationConfig.quickActions ?? []).map((definition) => [definition.id, definition]),
);

/**
 * The create shortcuts a person chose, in their order, minus the ones they
 * cannot perform or whose part of the product the organization has turned off.
 * Every surface that offers quick actions reads from here so they all agree.
 */
export function useSidebarQuickActions(): SidebarQuickAction[] {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const capabilities = useOrgCapabilities();
  const { data: preferences } = useSidebarPreferences();
  const quickActionIds = preferences?.quickActionIds;

  return useMemo(
    () =>
      (quickActionIds ?? [])
        .map((id) => QUICK_ACTIONS_BY_ID.get(id))
        .filter((definition): definition is QuickActionCommand => {
          if (!definition || !QUICK_ACTION_ICONS[definition.id]) {
            return false;
          }
          return canAccessQuickAction(definition, { capabilities, hasPermission });
        })
        .map((definition) => ({
          definition,
          icon: QUICK_ACTION_ICONS[definition.id],
          href: buildCommandHref(definition.path, definition.query),
        })),
    [capabilities, hasPermission, quickActionIds],
  );
}
