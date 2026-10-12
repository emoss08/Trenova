import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";

/** A signed-in person's permission manifest holding exactly these grants, by resource. */
export function permissionManifest(permissions: Record<string, number>): PermissionManifest {
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
    permissions,
    routeAccess: {},
    availableOrgs: [],
    checksum: "test",
    expiresAt: 1_782_403_304,
  } as PermissionManifest;
}

/** Signs a test's person in with exactly these grants, by resource. */
export function seedPermissions(permissions: Record<string, number>): void {
  usePermissionStore.setState({
    manifest: permissionManifest(permissions),
    lastFetched: Date.now(),
    isLoading: false,
  });
}

/** Signs the test's person out again, so no grant leaks into the next test. */
export function clearPermissions(): void {
  usePermissionStore.getState().clearPermissions();
}
