import { createCapabilityLoader } from "@/lib/route-permission";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import type { LoaderFunction, LoaderFunctionArgs } from "react-router";
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

async function run(loader: LoaderFunction): Promise<unknown> {
  return await loader({
    request: new Request("http://localhost/dispatch/carriers"),
    params: {},
    context: undefined,
  } as unknown as LoaderFunctionArgs);
}

describe("createCapabilityLoader", () => {
  afterEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("lets the route through when the organization has the capability", async () => {
    signIn(true);

    await expect(run(createCapabilityLoader(OrganizationCapability.Brokerage))).resolves.toBeNull();
  });

  it("throws the same 403 Response shape the permission loader throws", async () => {
    signIn(false);

    const thrown: unknown = await run(createCapabilityLoader(OrganizationCapability.Brokerage))
      .then(() => null)
      .catch((error: unknown) => error);

    if (!(thrown instanceof Response)) {
      throw new Error(`expected a thrown Response, received ${String(thrown)}`);
    }

    expect(thrown.status).toBe(403);
    expect(thrown.statusText).toBe("Brokerage not enabled");
    await expect(thrown.text()).resolves.toBe("Brokerage is not enabled for this organization");
  });

  it("does not gate a capability the organization still has", async () => {
    signIn(false);

    await expect(
      run(createCapabilityLoader(OrganizationCapability.AssetOperations)),
    ).resolves.toBeNull();
  });

  it("fails open when there is no signed-in user yet", async () => {
    useAuthStore.setState({ user: null, isAuthenticated: false });

    await expect(run(createCapabilityLoader(OrganizationCapability.Brokerage))).resolves.toBeNull();
  });
});
