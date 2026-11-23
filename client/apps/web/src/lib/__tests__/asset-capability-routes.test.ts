import { routeAt } from "@/test/route-tree";
import type { RouteGateContext } from "@trenova/shared/lib/router/gates";
import { enterContext } from "@trenova/shared/testing/gate-context";
import { RouteRefusal } from "@trenova/shared/lib/router/refusal";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

const ALL_OPERATIONS = 0xffff;
const CAPABILITY_STATUS_TEXT = "Asset Operations not enabled";

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

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
    unavailableFeatures: [],
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
    checkAuth: async () => true,
    checkSession: async () => ({ status: "signed_in" }),
  } as unknown as Parameters<typeof useAuthStore.setState>[0]);

  usePermissionStore.setState({ manifest: manifest(), lastFetched: Date.now(), isLoading: false });
}

// Runs the route's gates; resolves when they let the person in and rejects with what they threw.
async function runGates(path: string): Promise<void> {
  const beforeLoad = routeAt(path).options.beforeLoad as
    | ((context: RouteGateContext) => unknown)
    | undefined;
  await beforeLoad?.(enterContext(path));
}

async function capabilityRefusal(path: string): Promise<RouteRefusal | null> {
  const thrown: unknown = await runGates(path)
    .then(() => null)
    .catch((error: unknown) => error);
  if (thrown instanceof RouteRefusal && thrown.statusText === CAPABILITY_STATUS_TEXT) {
    return thrown;
  }
  return null;
}

/** Routes that only make sense for an organization that owns trucks and employs drivers. */
const ASSET_ONLY_ROUTES = [
  "/hr/workers",
  "/hr/workers/$id",
  "/hr/workers/new",
  "/hr/credential-types",
  "/hr/training-courses",
  "/hr/checklist-templates",
  "/hr/review-templates",
  "/hr/pto-policies",
  "/hr/holidays",
  "/equipment/tractors",
  "/equipment/trailers",
  "/dispatch/configuration-files/fleet-codes",
  "/payroll/workspace",
  "/payroll/settlements",
  "/payroll/disputes",
  "/payroll/expenses",
  "/payroll/settlement-batches",
  "/payroll/pay-events",
  "/payroll/pay-profiles",
  "/payroll/deductions",
  "/payroll/earnings",
  "/payroll/pay-codes",
  "/payroll/advances",
  "/payroll/escrow-accounts",
  "/admin/settlement-control",
  "/admin/dash-control",
];

/**
 * Equipment types describe the trailer a load needs, not one the organization
 * owns, and dispatch controls keeps its page so the neutral settings on it stay
 * editable — the driver-compliance fields are hidden inside the form instead.
 */
const NEVER_GATED_ROUTES = [
  "/equipment/configuration-files/equipment-types",
  "/equipment/configuration-files/equipment-manufacturers",
  "/dispatch/configuration-files/location-categories",
  "/admin/dispatch-controls",
  "/dispatch/console",
];

describe("asset-only routes", () => {
  beforeEach(() => {
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  afterEach(() => {
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("refuses every asset-only route for a brokerage", async () => {
    signIn(brokerageOnly);

    for (const path of ASSET_ONLY_ROUTES) {
      const refusal = await capabilityRefusal(path);
      expect(refusal, `${path} should be refused`).not.toBeNull();
      expect(refusal?.status).toBe(403);
      expect(refusal?.data).toBe("Asset Operations is not enabled for this organization");
    }
  });

  it("lets an asset organization through every one of them", async () => {
    signIn(hybrid);

    for (const path of ASSET_ONLY_ROUTES) {
      await expect(runGates(path), `${path} should load`).resolves.toBeUndefined();
    }
  });

  it("leaves freight-requirement and neutral routes open to a brokerage", async () => {
    signIn(brokerageOnly);

    for (const path of NEVER_GATED_ROUTES) {
      await expect(runGates(path), `${path} should stay open`).resolves.toBeUndefined();
    }
  });

  it("checks the capability before the permission, so a refusal names the capability", async () => {
    signIn(brokerageOnly);
    usePermissionStore.setState({ manifest: null, lastFetched: null, isLoading: false });

    const thrown: unknown = await runGates("/payroll/workspace")
      .then(() => null)
      .catch((error: unknown) => error);

    expect(thrown).toBeInstanceOf(RouteRefusal);
    expect((thrown as RouteRefusal).statusText).toBe(CAPABILITY_STATUS_TEXT);
  });
});
