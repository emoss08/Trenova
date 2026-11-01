import {
  MyPermissionVersionDocument,
  MyPermissionsDocument,
} from "@trenova/graphql/generated/graphql";
import type { PermissionManifest, PermissionVersion } from "@trenova/shared/types/permission";
import { requestGraphQL } from "./graphql";

/**
 * The caller's permission manifest. GraphQL carries the per-resource masks and screen access
 * as lists; the store keeps them as records keyed by resource and route.
 */
export async function getPermissionManifest(): Promise<PermissionManifest> {
  const data = await requestGraphQL({
    document: MyPermissionsDocument,
    operationName: "MyPermissions",
  });
  const manifest = data.myPermissions;

  return {
    version: manifest.version,
    userId: manifest.userId,
    organizationId: manifest.organizationId,
    activeRoleIds: manifest.activeRoleIds,
    authorizedRoleIds: manifest.authorizedRoleIds,
    activeRoles: manifest.activeRoles.map((role) => ({ ...role })),
    authorizedRoles: manifest.authorizedRoles.map((role) => ({ ...role })),
    requiresRoleActivation: manifest.requiresRoleActivation,
    maxSensitivity: manifest.maxSensitivity as PermissionManifest["maxSensitivity"],
    permissions: Object.fromEntries(
      manifest.permissions.map((grant) => [grant.resource, grant.operations]),
    ),
    routeAccess: Object.fromEntries(
      manifest.routeAccess.map((entry) => [entry.route, entry.allowed]),
    ),
    availableOrgs: manifest.availableOrgs.map((org) => ({ id: org.id, name: org.name })),
    checksum: manifest.checksum,
    expiresAt: manifest.expiresAt,
  };
}

export async function getPermissionVersion(): Promise<PermissionVersion> {
  const data = await requestGraphQL({
    document: MyPermissionVersionDocument,
    operationName: "MyPermissionVersion",
  });
  return {
    checksum: data.myPermissionVersion.checksum,
    expiresAt: data.myPermissionVersion.expiresAt,
  };
}
