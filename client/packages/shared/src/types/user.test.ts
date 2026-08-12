import { describe, expect, it } from "vitest";
import {
  loginResponseSchema,
  userOrganizationMembershipSchema,
} from "@trenova/shared/types/user";

describe("loginResponseSchema", () => {
  it("normalizes nullable role ids from login responses", () => {
    const parsed = loginResponseSchema.parse({
      user: {
        id: "usr_01KSJG9AG6XFQ8HDX5309293A9",
        businessUnitId: "bu_01KSJG9ADGK726S3YXAWMWQ2G3",
        currentOrganizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
        status: "Active",
        name: "System Administrator",
        username: "admin",
        timeFormat: "12-hour",
        emailAddress: "admin@trenova.app",
        profilePicUrl: "",
        thumbnailUrl: "",
        timezone: "America/Los_Angeles",
        isLocked: false,
        mustChangePassword: false,
        version: 0,
        createdAt: 1779811265,
        updatedAt: 1779811264,
        lastLoginAt: 1779811303,
      },
      expiresAt: 1782403304,
      sessionId: "ses_01KSJGAH46DYVJVZ31M8M4NE4F",
      csrfToken: "XSDj7-qiHSIUxNMki28caJ_32JJiFXoOBBtnIKFTMcc",
      activeRoleIds: null,
      authorizedRoleIds: ["rol_01KSJG9AYDWVSQESCYW4855C01"],
      activeRoles: null,
      authorizedRoles: [
        {
          id: "rol_01KSJG9AYDWVSQESCYW4855C01",
          name: "Dispatcher",
          description: null,
          isSystem: false,
        },
      ],
      requiresRoleActivation: true,
    });

    expect(parsed.activeRoleIds).toEqual([]);
    expect(parsed.authorizedRoleIds).toEqual(["rol_01KSJG9AYDWVSQESCYW4855C01"]);
    expect(parsed.activeRoles).toEqual([]);
    expect(parsed.authorizedRoles[0]?.name).toBe("Dispatcher");
    expect(parsed.authorizedRoles[0]?.description).toBe("");
  });
});

describe("userOrganizationMembershipSchema organization capabilities", () => {
  const membership = {
    id: "uom_01KSJG9AG6XFQ8HDX5309293A9",
    userId: "usr_01KSJG9AG6XFQ8HDX5309293A9",
    businessUnitId: "bu_01KSJG9ADGK726S3YXAWMWQ2G3",
    organizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
    isDefault: true,
  };

  it("fails open to enabled when a stale payload omits the capability flags", () => {
    const parsed = userOrganizationMembershipSchema.parse({
      ...membership,
      organization: {
        id: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
        name: "Trenova Logistics",
        city: "Dallas",
        state: { id: "us_01", name: "Texas" },
      },
    });

    expect(parsed.organization?.brokerageEnabled).toBe(true);
    expect(parsed.organization?.assetOperationsEnabled).toBe(true);
  });

  it("fails open to enabled when the flags arrive as null", () => {
    const parsed = userOrganizationMembershipSchema.parse({
      ...membership,
      organization: {
        id: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
        name: "Trenova Logistics",
        brokerageEnabled: null,
        assetOperationsEnabled: null,
      },
    });

    expect(parsed.organization?.brokerageEnabled).toBe(true);
    expect(parsed.organization?.assetOperationsEnabled).toBe(true);
  });

  it("keeps an explicitly disabled capability disabled", () => {
    const parsed = userOrganizationMembershipSchema.parse({
      ...membership,
      organization: {
        id: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
        name: "Asset Carrier Co",
        brokerageEnabled: false,
        assetOperationsEnabled: true,
      },
    });

    expect(parsed.organization?.brokerageEnabled).toBe(false);
    expect(parsed.organization?.assetOperationsEnabled).toBe(true);
  });
});
