import type { OrganizationSettings } from "@trenova/shared/types/organization";

/**
 * An organization as `OrganizationSettings` returns it (organization/page.graphql): every
 * field the form edits holds a value the server's validator accepts.
 */
export const savedOrganization: OrganizationSettings = {
  id: "org_01",
  version: 3,
  createdAt: 1_700_000_000,
  updatedAt: 1_700_000_000,
  bucketName: "org-01",
  businessUnitId: "bu_01",
  loginSlug: "acme",
  name: "Acme Freight",
  scacCode: "ACMF",
  dotNumber: "1234567",
  logoUrl: "",
  iconUrl: "",
  addressLine1: "1 Main St",
  addressLine2: "Suite 200",
  city: "Columbus",
  stateId: "us_oh",
  postalCode: "43004",
  timezone: "America/New_York",
  taxId: "12-3456789",
  brokerageEnabled: true,
  assetOperationsEnabled: true,
  state: { id: "us_oh", name: "Ohio", abbreviation: "OH", countryName: "United States" },
};
