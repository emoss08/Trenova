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
    expect(parsed.error?.issues.map((issue) => issue.path.join("."))).toEqual([
      "addressLine1",
      "postalCode",
    ]);
  });

  const complete = {
    ...managedInstanceOrganization,
    scacCode: "ACMF",
    dotNumber: "1234567",
    addressLine1: "1 Main St",
    city: "Columbus",
    postalCode: "43004",
    loginSlug: "acme-freight",
  };

  function refusedPaths(change: Record<string, unknown>) {
    const parsed = organizationSettingsSchema.safeParse({ ...complete, ...change });
    return parsed.success ? [] : parsed.error.issues.map((issue) => issue.path.join("."));
  }

  it("accepts a complete organization", () => {
    expect(refusedPaths({})).toEqual([]);
  });

  // The server's validator refuses each of these; the form refuses them first, on the field.
  it.each([
    ["a placeholder SCAC code in any case", { scacCode: "tbdx" }, ["scacCode"]],
    ["a SCAC code that is not four characters", { scacCode: "ACM" }, ["scacCode"]],
    ["a DOT number with letters", { dotNumber: "12A4" }, ["dotNumber"]],
    ["a DOT number past eight digits", { dotNumber: "123456789" }, ["dotNumber"]],
    ["the placeholder DOT number", { dotNumber: "0" }, ["dotNumber"]],
    ["the placeholder city", { city: "Pending" }, ["city"]],
    ["the placeholder ZIP code", { postalCode: "00000" }, ["postalCode"]],
    ["a sign-in address with capitals", { loginSlug: "Acme" }, ["loginSlug"]],
    ["a sign-in address with a trailing hyphen", { loginSlug: "acme-" }, ["loginSlug"]],
    ["a name past 100 characters", { name: "a".repeat(101) }, ["name"]],
    ["an address line past 150 characters", { addressLine1: "a".repeat(151) }, ["addressLine1"]],
  ])("refuses %s", (_, change, paths) => {
    expect(refusedPaths(change)).toEqual(paths);
  });

  it("lets the sign-in address be left empty", () => {
    expect(refusedPaths({ loginSlug: "" })).toEqual([]);
    expect(refusedPaths({ loginSlug: null })).toEqual([]);
  });

  it("refuses an organization that neither brokers nor runs assets, on both switches", () => {
    expect(refusedPaths({ brokerageEnabled: false, assetOperationsEnabled: false })).toEqual([
      "brokerageEnabled",
      "assetOperationsEnabled",
    ]);
    expect(refusedPaths({ brokerageEnabled: false, assetOperationsEnabled: true })).toEqual([]);
  });
});
