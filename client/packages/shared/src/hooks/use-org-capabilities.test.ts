import { describe, expect, it } from "vitest";
import {
  hasOrganizationCapability,
  OrganizationCapability,
  organizationCapabilityLabel,
  organizationCapabilityPresetValues,
  resolveOrganizationCapabilityPreset,
} from "@trenova/shared/types/organization-capability";
import {
  ORGANIZATION_CAPABILITIES_ENABLED,
  resolveOrgCapabilities,
} from "@trenova/shared/hooks/use-org-capabilities";
import type { User } from "@trenova/shared/types/user";

function userWithMemberships(
  currentOrganizationId: string,
  memberships: {
    organizationId: string;
    brokerageEnabled: boolean;
    assetOperationsEnabled: boolean;
  }[],
): User {
  return {
    currentOrganizationId,
    memberships: memberships.map((membership) => ({
      userId: "usr_01",
      organizationId: membership.organizationId,
      isDefault: false,
      organization: {
        id: membership.organizationId,
        name: membership.organizationId,
        brokerageEnabled: membership.brokerageEnabled,
        assetOperationsEnabled: membership.assetOperationsEnabled,
      },
    })),
  } as unknown as User;
}

describe("resolveOrgCapabilities", () => {
  it("reads the capabilities of the membership matching the current organization", () => {
    const user = userWithMemberships("org_b", [
      { organizationId: "org_a", brokerageEnabled: true, assetOperationsEnabled: true },
      { organizationId: "org_b", brokerageEnabled: false, assetOperationsEnabled: true },
    ]);

    expect(resolveOrgCapabilities(user)).toEqual({
      brokerageEnabled: false,
      assetOperationsEnabled: true,
    });
  });

  it("fails open when there is no signed-in user", () => {
    expect(resolveOrgCapabilities(null)).toEqual(ORGANIZATION_CAPABILITIES_ENABLED);
  });

  it("fails open when no membership matches the current organization", () => {
    const user = userWithMemberships("org_missing", [
      { organizationId: "org_a", brokerageEnabled: false, assetOperationsEnabled: false },
    ]);

    expect(resolveOrgCapabilities(user)).toEqual(ORGANIZATION_CAPABILITIES_ENABLED);
  });

  it("fails open when the membership carries no nested organization projection", () => {
    const user = {
      currentOrganizationId: "org_a",
      memberships: [{ userId: "usr_01", organizationId: "org_a", isDefault: true }],
    } as unknown as User;

    expect(resolveOrgCapabilities(user)).toEqual(ORGANIZATION_CAPABILITIES_ENABLED);
  });
});

describe("hasOrganizationCapability", () => {
  it("maps each capability onto its own flag", () => {
    const capabilities = { brokerageEnabled: false, assetOperationsEnabled: true };

    expect(hasOrganizationCapability(capabilities, OrganizationCapability.Brokerage)).toBe(false);
    expect(hasOrganizationCapability(capabilities, OrganizationCapability.AssetOperations)).toBe(
      true,
    );
  });

  it("labels capabilities for user-facing messages", () => {
    expect(organizationCapabilityLabel(OrganizationCapability.Brokerage)).toBe("Brokerage");
    expect(organizationCapabilityLabel(OrganizationCapability.AssetOperations)).toBe(
      "Asset Operations",
    );
  });
});

describe("organization capability presets", () => {
  it("maps each preset onto the pair of flags it sets", () => {
    expect(organizationCapabilityPresetValues("asset")).toEqual({
      brokerageEnabled: false,
      assetOperationsEnabled: true,
    });
    expect(organizationCapabilityPresetValues("brokerage")).toEqual({
      brokerageEnabled: true,
      assetOperationsEnabled: false,
    });
    expect(organizationCapabilityPresetValues("hybrid")).toEqual({
      brokerageEnabled: true,
      assetOperationsEnabled: true,
    });
  });

  it("names the preset a pair of flags corresponds to", () => {
    expect(
      resolveOrganizationCapabilityPreset({
        brokerageEnabled: false,
        assetOperationsEnabled: true,
      }),
    ).toBe("asset");
    expect(
      resolveOrganizationCapabilityPreset({
        brokerageEnabled: true,
        assetOperationsEnabled: false,
      }),
    ).toBe("brokerage");
    expect(
      resolveOrganizationCapabilityPreset({
        brokerageEnabled: true,
        assetOperationsEnabled: true,
      }),
    ).toBe("hybrid");
  });

  it("calls a combination matching no preset custom rather than forcing one", () => {
    expect(
      resolveOrganizationCapabilityPreset({
        brokerageEnabled: false,
        assetOperationsEnabled: false,
      }),
    ).toBe("custom");
  });
});
