import { requireCapability } from "@/lib/route-permission";
import type { RouteGate } from "@trenova/shared/lib/router/gates";
import { RouteRefusal } from "@trenova/shared/lib/router/refusal";
import { enterContext } from "@trenova/shared/testing/gate-context";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it } from "vitest";

function signIn(brokerageEnabled: boolean) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: {
            id: "org_01",
            name: "Asset Carrier Co",
            brokerageEnabled,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

async function run(gate: RouteGate): Promise<void> {
  await gate(enterContext("/dispatch/carriers"));
}

describe("requireCapability", () => {
  afterEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("lets the route through when the organization has the capability", async () => {
    signIn(true);

    await expect(run(requireCapability(OrganizationCapability.Brokerage))).resolves.toBeUndefined();
  });

  it("throws the same 403 refusal the permission gate throws", async () => {
    signIn(false);

    const thrown: unknown = await run(requireCapability(OrganizationCapability.Brokerage))
      .then(() => null)
      .catch((error: unknown) => error);

    if (!(thrown instanceof RouteRefusal)) {
      throw new Error(`expected a thrown RouteRefusal, received ${String(thrown)}`);
    }

    expect(thrown.status).toBe(403);
    expect(thrown.statusText).toBe("Brokerage not enabled");
    expect(thrown.data).toBe("Brokerage is not enabled for this organization");
  });

  it("does not gate a capability the organization still has", async () => {
    signIn(false);

    await expect(
      run(requireCapability(OrganizationCapability.AssetOperations)),
    ).resolves.toBeUndefined();
  });

  it("fails open when there is no signed-in user yet", async () => {
    useAuthStore.setState({ user: null, isAuthenticated: false });

    await expect(run(requireCapability(OrganizationCapability.Brokerage))).resolves.toBeUndefined();
  });
});
