import { adminLinks, navigationConfig } from "@/config/navigation.config";
import { isNavGroup, type NavGroup, type NavItem } from "@/config/navigation.types";
import { useAccessibleAdminLinks } from "@/hooks/use-accessible-admin-links";
import {
  canAccessNavEntry,
  canAccessQuickAction,
  filterNavModules,
  type NavAccessContext,
} from "@/hooks/use-filtered-navigation";
import { renderHook } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it } from "vitest";

const ALL_OPERATIONS = 0xffff;

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

/**
 * Every permission is granted in both directions, so the only thing that can
 * move an entry in or out of these expectations is the capability itself.
 */
function context(capabilities: OrganizationCapabilities): NavAccessContext {
  return { capabilities, hasPermission: () => true };
}

function collectEntryIds(entries: (NavItem | NavGroup)[]): string[] {
  return entries.flatMap((entry) =>
    isNavGroup(entry) ? [entry.id, ...entry.items.map((item) => item.id)] : [entry.id],
  );
}

function visibleIds(capabilities: OrganizationCapabilities): Set<string> {
  const modules = filterNavModules(navigationConfig.modules, context(capabilities));
  return new Set(modules.flatMap((module) => [module.id, ...collectEntryIds(module.navigation)]));
}

/** Nav entries an organization without its own trucks has no use for. */
const ASSET_ONLY_NAV_IDS = [
  "workers",
  "tractors",
  "trailers",
  "fleet-codes",
  "payroll",
  "settlement-workspace",
  "driver-settlements",
  "settlement-disputes",
  "driver-expenses",
  "settlement-batches",
  "pay-events",
  "payroll-config-group",
  "pay-profiles",
  "recurring-deductions",
  "recurring-earnings",
  "pay-codes",
  "pay-advances",
  "escrow-accounts",
];

/**
 * Equipment types and manufacturers describe what a load requires, not what the
 * organization owns — a brokerage still has to say "this needs a 53' reefer".
 */
const NEVER_GATED_NAV_IDS = [
  "equipment-types",
  "equipment-manufacturers",
  "dispatch-console",
  "locations",
  "location-categories",
  "carriers",
];

describe("asset-only navigation", () => {
  it("shows every asset entry to an organization that runs its own trucks", () => {
    const ids = visibleIds(hybrid);

    for (const id of [...ASSET_ONLY_NAV_IDS, ...NEVER_GATED_NAV_IDS]) {
      expect(ids.has(id), `${id} should be visible to an asset organization`).toBe(true);
    }
  });

  it("hides every asset entry from a brokerage", () => {
    const ids = visibleIds(brokerageOnly);

    for (const id of ASSET_ONLY_NAV_IDS) {
      expect(ids.has(id), `${id} should be hidden from a brokerage`).toBe(false);
    }
  });

  it("keeps freight-requirement configuration reachable for a brokerage", () => {
    const ids = visibleIds(brokerageOnly);

    for (const id of NEVER_GATED_NAV_IDS) {
      expect(ids.has(id), `${id} should stay visible to a brokerage`).toBe(true);
    }
  });

  it("drops the payroll module itself rather than emptying it out", () => {
    const modules = filterNavModules(navigationConfig.modules, context(brokerageOnly));

    expect(modules.some((module) => module.id === "payroll")).toBe(false);
  });

  it("gates the payroll module at the module level, as carrier settlements is", () => {
    const payroll = navigationConfig.modules.find((module) => module.id === "payroll");
    if (!payroll) {
      throw new Error("payroll module missing from navigation config");
    }

    expect(canAccessNavEntry(payroll, context(brokerageOnly))).toBe(false);
    expect(canAccessNavEntry(payroll, context(hybrid))).toBe(true);
  });
});

const ASSET_ONLY_QUICK_ACTION_IDS = [
  "create-worker",
  "create-tractor",
  "create-trailer",
  "create-fleet-code",
];

const NEVER_GATED_QUICK_ACTION_IDS = [
  "create-equipment-type",
  "create-equipment-manufacturer",
  "create-location",
];

describe("asset-only quick actions", () => {
  function quickAction(id: string) {
    const action = (navigationConfig.quickActions ?? []).find((entry) => entry.id === id);
    if (!action) {
      throw new Error(`quick action ${id} missing from navigation config`);
    }
    return action;
  }

  it("offers every create action to an asset organization", () => {
    for (const id of [...ASSET_ONLY_QUICK_ACTION_IDS, ...NEVER_GATED_QUICK_ACTION_IDS]) {
      expect(canAccessQuickAction(quickAction(id), context(hybrid)), id).toBe(true);
    }
  });

  it("withholds the asset create actions from a brokerage", () => {
    for (const id of ASSET_ONLY_QUICK_ACTION_IDS) {
      expect(canAccessQuickAction(quickAction(id), context(brokerageOnly)), id).toBe(false);
    }
  });

  it("keeps freight-requirement create actions available to a brokerage", () => {
    for (const id of NEVER_GATED_QUICK_ACTION_IDS) {
      expect(canAccessQuickAction(quickAction(id), context(brokerageOnly)), id).toBe(true);
    }
  });

  it("still refuses a quick action the user lacks the permission for", () => {
    expect(
      canAccessQuickAction(quickAction("create-worker"), {
        capabilities: hybrid,
        hasPermission: () => false,
      }),
    ).toBe(false);
  });
});

function manifest(): PermissionManifest {
  return {
    version: "1.0",
    userId: "usr_01",
    organizationId: "org_01",
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
    maxSensitivity: "internal",
    permissions: Object.fromEntries(
      Object.values(Resource).map((resource) => [resource, ALL_OPERATIONS]),
    ),
    routeAccess: {},
    availableOrgs: [],
    checksum: "abc123",
    expiresAt: 1_782_403_304,
  } as unknown as PermissionManifest;
}

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Brokerage Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });

  usePermissionStore.setState({ manifest: manifest(), lastFetched: Date.now(), isLoading: false });
}

describe("asset-only admin links", () => {
  afterEach(() => {
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("lists the driver settlement and driver portal controls for an asset organization", () => {
    signIn(hybrid);

    const { result } = renderHook(() => useAccessibleAdminLinks());
    const hrefs = result.current.map((link) => link.href);

    expect(hrefs).toContain("/admin/settlement-control");
    expect(hrefs).toContain("/admin/dash-control");
  });

  it("withholds them from a brokerage while keeping the neutral controls", () => {
    signIn(brokerageOnly);

    const { result } = renderHook(() => useAccessibleAdminLinks());
    const hrefs = result.current.map((link) => link.href);

    expect(hrefs).not.toContain("/admin/settlement-control");
    expect(hrefs).not.toContain("/admin/dash-control");
    expect(hrefs).toContain("/admin/accounting-control");
    expect(hrefs).toContain("/admin/organization-settings");
  });

  it("keeps every admin link declared in the config reachable when both halves run", () => {
    signIn(hybrid);

    const { result } = renderHook(() => useAccessibleAdminLinks());

    expect(result.current).toHaveLength(adminLinks.filter((link) => !link.disabled).length);
  });
});
