import { navigationConfig } from "@/config/navigation.config";
import {
  isNavGroup,
  type NavGroup,
  type NavItem,
  type NavModule,
} from "@/config/navigation.types";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import {
  hasOrganizationCapability,
  type OrganizationCapabilities,
} from "@trenova/shared/types/organization-capability";
import { Operation, type OperationType } from "@trenova/shared/types/permission";
import { useMemo } from "react";

export interface NavAccessContext {
  hasPermission: (resource: string, operation: OperationType) => boolean;
  capabilities: OrganizationCapabilities;
}

/**
 * The single predicate every navigation surface funnels through. A capability
 * only ever removes an entry the organization does not use; permissions still
 * decide what an authorized user may reach.
 */
export function canAccessNavEntry(
  entry: NavItem | NavGroup | NavModule,
  context: NavAccessContext,
): boolean {
  if (entry.capability && !hasOrganizationCapability(context.capabilities, entry.capability)) {
    return false;
  }

  if (entry.resource) {
    return context.hasPermission(entry.resource, Operation.Read);
  }

  return true;
}

export function filterNavEntries(
  entries: (NavItem | NavGroup)[],
  context: NavAccessContext,
): (NavItem | NavGroup)[] {
  return entries
    .map((entry) => {
      if (isNavGroup(entry)) {
        if (!canAccessNavEntry(entry, context)) {
          return null;
        }
        const items = entry.items.filter((item) => canAccessNavEntry(item, context));
        if (items.length === 0) {
          return null;
        }
        return { ...entry, items };
      }
      return canAccessNavEntry(entry, context) ? entry : null;
    })
    .filter((entry): entry is NavItem | NavGroup => entry !== null);
}

export function filterNavModules(modules: NavModule[], context: NavAccessContext): NavModule[] {
  return modules
    .filter((module) => canAccessNavEntry(module, context))
    .map((module) => ({
      ...module,
      navigation: filterNavEntries(module.navigation, context),
    }))
    .filter((module) => {
      if (module.hideSecondarySidebar) return true;
      return module.navigation.length > 0;
    });
}

const PERMIT_ALL = () => true;

function useNavAccessContext(): NavAccessContext {
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const capabilities = useOrgCapabilities();

  return useMemo(
    // Before the manifest lands nothing is known about permissions, so they are
    // treated as granted rather than flashing a stripped-down menu. Capabilities
    // ride the session payload and are known immediately, so they still apply.
    () => ({ hasPermission: manifest ? hasPermission : PERMIT_ALL, capabilities }),
    [manifest, hasPermission, capabilities],
  );
}

export function useFilteredNavigation() {
  const context = useNavAccessContext();

  return useMemo(() => filterNavModules(navigationConfig.modules, context), [context]);
}

export function useFilteredModuleNavigation(module: NavModule | null) {
  const context = useNavAccessContext();

  return useMemo(() => {
    if (!module) {
      return [];
    }

    return filterNavEntries(module.navigation, context);
  }, [module, context]);
}
