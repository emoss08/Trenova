import { beforeEach, describe, expect, it, vi } from "vitest";
import { getOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { organization } from "./organization";

const organizationServiceMocks = vi.hoisted(() => ({
  getLogoURL: vi.fn(),
  getMicrosoftSSOConfig: vi.fn(),
  getOktaSSOConfig: vi.fn(),
  getTenantLoginMetadata: vi.fn(),
  listAccessPolicies: vi.fn(),
  listAuthEvents: vi.fn(),
  listExternalIdentities: vi.fn(),
  listIdentityProviders: vi.fn(),
  listMFAAuthenticators: vi.fn(),
  listProvisioningAudit: vi.fn(),
  listRiskDecisions: vi.fn(),
  listSCIMDirectories: vi.fn(),
}));

vi.mock("@/lib/graphql/organization", () => ({
  getOrganizationSettingsGraphQL: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    organizationService: organizationServiceMocks,
  },
}));

const getOrganizationSettingsGraphQLMock = vi.mocked(getOrganizationSettingsGraphQL);

describe("organization query keys", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses GraphQL for organization settings detail while preserving the query key", async () => {
    getOrganizationSettingsGraphQLMock.mockResolvedValueOnce({ id: "org_1" } as never);

    const query = organization.detail("org_1");
    const response = await query.queryFn({
      queryKey: query.queryKey,
      signal: new AbortController().signal,
    } as unknown as Parameters<typeof query.queryFn>[0]);

    expect(query.queryKey).toEqual(["organization", "detail", "detail", "org_1"]);
    expect(response).toEqual({ id: "org_1" });
    expect(getOrganizationSettingsGraphQLMock).toHaveBeenCalledWith("org_1");
  });
});
