import { organizationSettingsSchema } from "@trenova/shared/types/organization";
import { describe, expect, it } from "vitest";
import { organizationSettingsTabs } from "../organization-settings-tabs";

/** What the record carries that nobody edits: its identity, tenancy, storage and the state read. */
const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "businessUnitId",
  "bucketName",
  "state",
]);

describe("organization settings tabs", () => {
  it("puts every field the organization's settings hold on exactly one tab", () => {
    const placed = organizationSettingsTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(organizationSettingsSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers the profile, the branding and the operations, in that order", () => {
    expect(organizationSettingsTabs().map((tab) => [tab.value, tab.label])).toEqual([
      ["profile", "Profile"],
      ["branding", "Branding"],
      ["operations", "Operations"],
    ]);
  });

  it("keeps the registered address with the profile and the regulatory numbers with operations", () => {
    const fields = Object.fromEntries(
      organizationSettingsTabs().map((tab) => [tab.value, tab.fields]),
    );

    expect(fields.profile).toEqual(
      expect.arrayContaining(["addressLine1", "addressLine2", "city", "stateId", "postalCode"]),
    );
    expect(fields.operations).toEqual(expect.arrayContaining(["scacCode", "dotNumber", "taxId"]));
  });
});
