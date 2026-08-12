import { routes } from "@/router";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import type { LoaderFunctionArgs, RouteObject } from "react-router";
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
    // protectedLoader asks the store, not the network.
    checkAuth: async () => true,
  } as unknown as Parameters<typeof useAuthStore.setState>[0]);

  usePermissionStore.setState({ manifest: manifest(), lastFetched: Date.now(), isLoading: false });
}

function joinPath(parent: string, child: string): string {
  if (child.startsWith("/")) {
    return child;
  }
  return `${parent.replace(/\/$/, "")}/${child}`;
}

function collectLoaders(
  entries: RouteObject[],
  parent = "",
  found = new Map<string, RouteObject>(),
): Map<string, RouteObject> {
  for (const entry of entries) {
    const path = entry.path === undefined ? parent : joinPath(parent, entry.path);
    if (entry.path !== undefined) {
      found.set(path, entry);
    }
    if (entry.children) {
      collectLoaders(entry.children, path, found);
    }
  }
  return found;
}

const routesByPath = collectLoaders(routes);

async function runLoader(path: string): Promise<unknown> {
  const route = routesByPath.get(path);
  if (!route) {
    throw new Error(`route ${path} is not registered`);
  }
  if (typeof route.loader !== "function") {
    throw new Error(`route ${path} has no loader`);
  }

  return await route.loader({
    request: new Request(`http://localhost${path}`),
    params: {},
    context: undefined,
  } as unknown as LoaderFunctionArgs);
}

async function capabilityRefusal(path: string): Promise<Response | null> {
  const thrown: unknown = await runLoader(path)
    .then(() => null)
    .catch((error: unknown) => error);

  if (thrown instanceof Response && thrown.statusText === CAPABILITY_STATUS_TEXT) {
    return thrown;
  }
  return null;
}

/** Routes that only make sense for an organization that owns trucks and employs drivers. */
const ASSET_ONLY_ROUTES = [
  "/dispatch/workers",
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
      await expect(refusal?.text()).resolves.toBe(
        "Asset Operations is not enabled for this organization",
      );
    }
  });

  it("lets an asset organization through every one of them", async () => {
    signIn(hybrid);

    for (const path of ASSET_ONLY_ROUTES) {
      await expect(runLoader(path), `${path} should load`).resolves.toBeNull();
    }
  });

  it("leaves freight-requirement and neutral routes open to a brokerage", async () => {
    signIn(brokerageOnly);

    for (const path of NEVER_GATED_ROUTES) {
      await expect(runLoader(path), `${path} should stay open`).resolves.toBeNull();
    }
  });

  it("checks the capability before the permission, so a refusal names the capability", async () => {
    signIn(brokerageOnly);
    usePermissionStore.setState({ manifest: null, lastFetched: null, isLoading: false });

    const thrown: unknown = await runLoader("/payroll/workspace")
      .then(() => null)
      .catch((error: unknown) => error);

    expect(thrown).toBeInstanceOf(Response);
    expect((thrown as Response).statusText).toBe(CAPABILITY_STATUS_TEXT);
  });
});
