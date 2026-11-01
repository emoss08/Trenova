import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationService } from "../organization";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);
const service = new OrganizationService();

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

const scimToken = {
  id: "sct_1",
  organizationId: "org_1",
  directoryId: "scd_1",
  name: "Okta",
  prefix: "scim_ab",
  status: "active",
  lastUsedAt: 0,
  expiresAt: 0,
  createdAt: 1_700_000_000,
  updatedAt: 1_700_000_000,
};

describe("OrganizationService", () => {
  it("returns a new SCIM token with its secret beside it, shown this once", async () => {
    requestGraphQLMock.mockResolvedValue({
      createSCIMToken: { token: "scim_ab.secret", scimToken },
    } as never);

    const issued = await service.createSCIMToken("scd_1", "Okta");

    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "CreateSCIMToken",
        variables: { directoryId: "scd_1", input: { name: "Okta" } },
      }),
    );
    expect(issued.token).toBe("scim_ab.secret");
    expect(issued.prefix).toBe("scim_ab");
  });

  it("reads the latest sign-ins from the auth event connection, newest first", async () => {
    requestGraphQLMock.mockResolvedValue({
      authEvents: {
        edges: [
          {
            node: {
              id: "aue_1",
              userId: null,
              identityProviderId: null,
              provider: "password",
              outcome: "denied",
              ipAddress: "10.0.0.1",
              userAgent: "",
              authenticatorAal: 1,
              federationFal: 1,
              mfaState: "",
              riskOutcome: "allow",
              riskSignals: [],
              errorCode: "bad_password",
              occurredAt: 1_700_000_100,
              createdAt: 1_700_000_100,
            },
          },
        ],
      },
    } as never);

    const events = await service.listAuthEvents();

    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "RecentAuthEvents", variables: { first: 10 } }),
    );
    expect(events).toHaveLength(1);
    expect(events[0]?.outcome).toBe("denied");
    expect(events[0]?.errorCode).toBe("bad_password");
  });

  it("keeps a stored client secret when the form leaves it empty", async () => {
    requestGraphQLMock.mockResolvedValue({
      upsertMicrosoftSsoConfig: {
        organizationId: "org_1",
        enabled: true,
        enforceSso: false,
        tenantId: "tenant",
        clientId: "client",
        redirectUrl: "https://app.example.com/callback",
        allowedDomains: ["example.com"],
        secretConfigured: true,
      },
    } as never);

    const saved = await service.upsertMicrosoftSSOConfig({
      organizationId: "org_1",
      enabled: true,
      enforceSso: false,
      tenantId: "tenant",
      clientId: "client",
      clientSecret: "",
      redirectUrl: "https://app.example.com/callback",
      allowedDomains: ["example.com"],
      secretConfigured: true,
    });

    expect(requestGraphQLMock.mock.calls[0]?.[0]).toMatchObject({
      operationName: "UpsertMicrosoftSSOConfig",
      variables: { input: { clientSecret: "", tenantId: "tenant" } },
    });
    expect(saved.secretConfigured).toBe(true);
    expect(saved.clientSecret).toBe("");
  });
});
