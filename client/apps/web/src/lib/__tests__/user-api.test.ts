import { beforeEach, describe, expect, it, vi } from "vitest";
import { getUserRoleAssignments, resetUserMFA, resetUserPassword } from "../user-api";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: mocks.requestGraphQL,
}));

describe("user-api", () => {
  beforeEach(() => {
    mocks.requestGraphQL.mockReset();
  });

  it("asks the server to mail a reset link to the person named", async () => {
    mocks.requestGraphQL.mockResolvedValue({ resetUserPassword: true });

    await resetUserPassword("usr_123");

    expect(mocks.requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "ResetUserPassword",
        variables: { userId: "usr_123" },
      }),
    );
  });

  it("resets the second factor of the person named", async () => {
    mocks.requestGraphQL.mockResolvedValue({ resetUserMfa: true });

    await resetUserMFA("usr_123");

    expect(mocks.requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "ResetUserMfa", variables: { userId: "usr_123" } }),
    );
  });

  it("returns a person's role assignments with the role they grant", async () => {
    mocks.requestGraphQL.mockResolvedValue({
      userRoleAssignments: [
        {
          id: "ura_1",
          userId: "usr_123",
          organizationId: "org_1",
          roleId: "rol_1",
          expiresAt: null,
          assignedBy: "usr_admin",
          assignedAt: 1_700_000_000,
          role: { id: "rol_1", name: "Dispatcher", isSystem: false },
        },
      ],
    });

    const assignments = await getUserRoleAssignments("usr_123");

    expect(assignments).toHaveLength(1);
    expect(assignments[0]?.roleId).toBe("rol_1");
    expect(assignments[0]?.role?.name).toBe("Dispatcher");
    expect(mocks.requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "UserRoleAssignments",
        variables: { userId: "usr_123" },
      }),
    );
  });
});
