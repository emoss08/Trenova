import { beforeEach, describe, expect, it, vi } from "vitest";
import { getPermissionManifest, getPermissionVersion } from "../permission-api";
import { requestGraphQL } from "../graphql";

vi.mock("../graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("getPermissionManifest", () => {
  it("turns the GraphQL lists back into the records the permission store reads", async () => {
    requestGraphQLMock.mockResolvedValue({
      myPermissions: {
        version: "2",
        userId: "usr_1",
        organizationId: "org_1",
        activeRoleIds: ["rol_1"],
        authorizedRoleIds: ["rol_1", "rol_2"],
        activeRoles: [
          {
            id: "rol_1",
            name: "Dispatcher",
            description: "Plans loads",
            isSystem: false,
            permissionCount: 3,
          },
        ],
        authorizedRoles: [],
        requiresRoleActivation: true,
        maxSensitivity: "restricted",
        permissions: [
          { resource: "carrier", operations: 1 },
          { resource: "shipment", operations: 7 },
        ],
        routeAccess: [
          { route: "/billing", allowed: false },
          { route: "/shipments", allowed: true },
        ],
        availableOrgs: [{ id: "org_1", name: "Acme" }],
        checksum: "abc",
        expiresAt: 1_800_000_000,
      },
    } as never);

    const manifest = await getPermissionManifest();

    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "MyPermissions" }),
    );
    expect(manifest).toEqual({
      version: "2",
      userId: "usr_1",
      organizationId: "org_1",
      activeRoleIds: ["rol_1"],
      authorizedRoleIds: ["rol_1", "rol_2"],
      activeRoles: [
        {
          id: "rol_1",
          name: "Dispatcher",
          description: "Plans loads",
          isSystem: false,
          permissionCount: 3,
        },
      ],
      authorizedRoles: [],
      requiresRoleActivation: true,
      maxSensitivity: "restricted",
      permissions: { carrier: 1, shipment: 7 },
      routeAccess: { "/billing": false, "/shipments": true },
      availableOrgs: [{ id: "org_1", name: "Acme" }],
      checksum: "abc",
      expiresAt: 1_800_000_000,
    });
  });
});

describe("getPermissionVersion", () => {
  it("returns the checksum the store compares its cached manifest against", async () => {
    requestGraphQLMock.mockResolvedValue({
      myPermissionVersion: { checksum: "def", expiresAt: 1_800_000_100 },
    } as never);

    await expect(getPermissionVersion()).resolves.toEqual({
      checksum: "def",
      expiresAt: 1_800_000_100,
    });
    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "MyPermissionVersion" }),
    );
  });
});
