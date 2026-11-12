import { describe, expect, it } from "vitest";
import { organizationSettingsSchema, storedOrganizationSettingsSchema } from "../organization";

const managedInstanceOrganization = {
  id: "org_01",
  version: 0,
  createdAt: 1794441464,
  updatedAt: 1794441464,
  name: "Trenova",
  scacCode: "ZZZZ",
  dotNumber: "0000000",
  addressLine1: "",
  city: "Unknown",
  stateId: "us_01",
  postalCode: "00000",
  timezone: "America/New_York",
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};

describe("organization settings", () => {
  it("reads an organization whose address nobody has filled in yet", () => {
    const parsed = storedOrganizationSettingsSchema.safeParse(managedInstanceOrganization);

    expect(parsed.success).toBe(true);
  });

  it("still asks the form for the address before saving", () => {
    const parsed = organizationSettingsSchema.safeParse(managedInstanceOrganization);

    expect(parsed.success).toBe(false);
    expect(parsed.error?.issues.map((issue) => issue.path.join("."))).toEqual(["addressLine1"]);
  });
});
