import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";

function manifest(permissions: Record<string, number>): PermissionManifest {
  return {
    version: "1.0",
    userId: "usr_01KSJG9AG6XFQ8HDX5309293A9",
    organizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
    maxSensitivity: "internal",
    permissions,
    routeAccess: { "/shipments": true },
    availableOrgs: [],
    checksum: "abc123",
    expiresAt: 1782403304,
  } as PermissionManifest;
}

describe("permission store fail-closed behaviour", () => {
  beforeEach(() => {
    usePermissionStore.setState({
      manifest: manifest({ [Resource.Shipment]: Operation.Read }),
      lastFetched: Date.now(),
      isLoading: false,
    });
  });

  afterEach(() => {
    usePermissionStore.getState().clearPermissions();
  });

  it("grants a permission the manifest actually carries", () => {
    expect(usePermissionStore.getState().hasPermission(Resource.Shipment, Operation.Read)).toBe(
      true,
    );
  });

  // The load-bearing assertion. The client resource enum is generated from Go, and for a
  // while it was missing eleven resources the server knows about. If an unrecognised key
  // resolved to "permitted", every such gap would have been a live authorization hole
  // rather than a feature nobody could gate. It must stay a denial.
  it("denies an unrecognised resource key rather than permitting it", () => {
    const { hasPermission, hasAnyPermission, hasAllPermissions } = usePermissionStore.getState();

    expect(hasPermission("resource_that_does_not_exist", Operation.Read)).toBe(false);
    expect(hasPermission("", Operation.Read)).toBe(false);
    expect(
      hasAnyPermission("resource_that_does_not_exist", [Operation.Read, Operation.Update]),
    ).toBe(false);
    expect(hasAllPermissions("resource_that_does_not_exist", [Operation.Read])).toBe(false);
  });

  it("denies a key that differs only in case from a known one", () => {
    expect(usePermissionStore.getState().hasPermission("API_KEY", Operation.Read)).toBe(false);
    expect(usePermissionStore.getState().hasPermission("apiKey", Operation.Read)).toBe(false);
  });

  it("does not let an inherited Object property masquerade as a grant", () => {
    const { hasPermission } = usePermissionStore.getState();

    expect(hasPermission("constructor", Operation.Read)).toBe(false);
    expect(hasPermission("toString", Operation.Read)).toBe(false);
    expect(hasPermission("__proto__", Operation.Read)).toBe(false);
  });

  it("denies a known resource for an operation its bitmask omits", () => {
    expect(usePermissionStore.getState().hasPermission(Resource.Shipment, Operation.Delete)).toBe(
      false,
    );
  });

  it("denies everything when no manifest has loaded", () => {
    usePermissionStore.getState().clearPermissions();

    const { hasPermission, canAccessRoute } = usePermissionStore.getState();
    expect(hasPermission(Resource.Shipment, Operation.Read)).toBe(false);
    expect(canAccessRoute("/shipments")).toBe(false);
  });

  it("denies an unrecognised route rather than permitting it", () => {
    const { canAccessRoute } = usePermissionStore.getState();

    expect(canAccessRoute("/shipments")).toBe(true);
    expect(canAccessRoute("/route-that-does-not-exist")).toBe(false);
    expect(canAccessRoute("")).toBe(false);
  });

  // The prototype-key cases above deny for the wrong reason under an undefined guard:
  // Function & operation coerces to 0. Put a real bitmask on the prototype and the
  // distinction becomes observable — an undefined guard lets it through to the bitmask,
  // where it PERMITS. Only an own-property check denies.
  it("denies an inherited key whose value would satisfy the bitmask", () => {
    const permissions = Object.create({
      inherited_resource: Operation.Read,
    }) as Record<string, number>;
    permissions[Resource.Shipment] = Operation.Read;

    usePermissionStore.setState({ manifest: manifest(permissions) });

    expect(permissions.inherited_resource).toBe(Operation.Read);
    expect(usePermissionStore.getState().hasPermission("inherited_resource", Operation.Read)).toBe(
      false,
    );
    expect(usePermissionStore.getState().hasPermission(Resource.Shipment, Operation.Read)).toBe(
      true,
    );
  });

  it("denies an inherited route whose value would satisfy the lookup", () => {
    const routeAccess = Object.create({ "/inherited-route": true }) as Record<string, boolean>;
    routeAccess["/shipments"] = true;

    usePermissionStore.setState({
      manifest: { ...manifest({}), routeAccess } as PermissionManifest,
    });

    expect(routeAccess["/inherited-route"]).toBe(true);
    expect(usePermissionStore.getState().canAccessRoute("/inherited-route")).toBe(false);
    expect(usePermissionStore.getState().canAccessRoute("/shipments")).toBe(true);
  });

  it("denies a resource present in the enum but absent from the manifest", () => {
    // Every resource the generator emits is a key the client may ask about; the manifest
    // only carries the ones this user was granted. The rest must read as denials.
    const granted = Resource.Shipment;
    const notGranted = Object.values(Resource).filter((value) => value !== granted);

    for (const resource of notGranted) {
      expect(usePermissionStore.getState().hasPermission(resource, Operation.Read)).toBe(false);
    }
  });
});
