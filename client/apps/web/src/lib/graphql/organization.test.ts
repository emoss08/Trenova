import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  OrganizationSettingsDocument,
  UpdateOrganizationSettingsDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import { getOrganizationSettingsGraphQL, updateOrganizationSettingsGraphQL } from "./organization";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

const organization: OrganizationSettings = {
  id: "org_1",
  version: 3,
  createdAt: 1_800_000_000,
  updatedAt: 1_800_000_100,
  bucketName: "acme-bucket",
  businessUnitId: "bu_1",
  loginSlug: "acme-logistics",
  name: "Acme Logistics",
  scacCode: "ACME",
  dotNumber: "1234567",
  logoUrl: "organization/logo/acme.png",
  addressLine1: "123 Main St",
  addressLine2: "Suite 200",
  city: "Chicago",
  stateId: "us_1",
  postalCode: "60601",
  timezone: "America/Chicago",
  taxId: "12-3456789",
  brokerageEnabled: true,
  assetOperationsEnabled: true,
  state: {
    id: "us_1",
    name: "Illinois",
    abbreviation: "IL",
  },
};

describe("organization GraphQL helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("requests and parses organization settings", async () => {
    requestGraphQLMock.mockResolvedValueOnce({ organization });

    const response = await getOrganizationSettingsGraphQL("org_1");

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: OrganizationSettingsDocument,
      operationName: "OrganizationSettings",
      variables: {
        id: "org_1",
        includeState: true,
        includeBu: false,
      },
    });
    expect(response).toEqual(organization);
  });

  it("maps organization settings updates to GraphQL input", async () => {
    requestGraphQLMock.mockResolvedValueOnce({ updateOrganization: organization });

    const response = await updateOrganizationSettingsGraphQL("org_1", organization);

    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: UpdateOrganizationSettingsDocument,
      operationName: "UpdateOrganizationSettings",
      variables: {
        id: "org_1",
        input: {
          version: 3,
          name: "Acme Logistics",
          loginSlug: "acme-logistics",
          scacCode: "ACME",
          dotNumber: "1234567",
          logoUrl: "organization/logo/acme.png",
          bucketName: "acme-bucket",
          addressLine1: "123 Main St",
          addressLine2: "Suite 200",
          city: "Chicago",
          stateId: "us_1",
          postalCode: "60601",
          timezone: "America/Chicago",
          taxId: "12-3456789",
          brokerageEnabled: true,
          assetOperationsEnabled: true,
        },
      },
    });
    expect(response).toEqual(organization);
  });

  it("carries a disabled brokerage capability through to the mutation input", async () => {
    const assetOnly: OrganizationSettings = {
      ...organization,
      brokerageEnabled: false,
      assetOperationsEnabled: true,
    };
    requestGraphQLMock.mockResolvedValueOnce({ updateOrganization: assetOnly });

    await updateOrganizationSettingsGraphQL("org_1", assetOnly);

    expect(requestGraphQLMock.mock.calls[0]?.[0]).toMatchObject({
      variables: {
        input: { brokerageEnabled: false, assetOperationsEnabled: true },
      },
    });
  });

  it("reads a response that predates the capability flags as fully enabled", async () => {
    const {
      brokerageEnabled: _brokerage,
      assetOperationsEnabled: _assets,
      ...legacy
    } = organization;
    requestGraphQLMock.mockResolvedValueOnce({ organization: legacy });

    const response = await getOrganizationSettingsGraphQL("org_1");

    expect(response.brokerageEnabled).toBe(true);
    expect(response.assetOperationsEnabled).toBe(true);
  });
});
