import { describe, expect, it } from "vitest";
import { permissionManifestSchema } from "./permission";

describe("permissionManifestSchema", () => {
  it("normalizes nullable role summary arrays", () => {
    const parsed = permissionManifestSchema.parse({
      version: "1.0",
      userId: "usr_01KSJG9AG6XFQ8HDX5309293A9",
      organizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
      isPlatformAdmin: false,
      isOrgAdmin: false,
      isBusinessUnitAdmin: false,
      activeRoleIds: null,
      authorizedRoleIds: null,
      activeRoles: null,
      authorizedRoles: null,
      requiresRoleActivation: false,
      maxSensitivity: "internal",
      permissions: {},
      routeAccess: {},
      availableOrgs: [],
      checksum: "abc123",
      expiresAt: 1782403304,
    });

    expect(parsed.activeRoleIds).toEqual([]);
    expect(parsed.authorizedRoleIds).toEqual([]);
    expect(parsed.activeRoles).toEqual([]);
    expect(parsed.authorizedRoles).toEqual([]);
  });
});
