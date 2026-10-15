import { describe, expect, it } from "vitest";
import {
  loginResponseSchema,
  userCreateSchema,
  userEditSchema,
  userOrganizationMembershipSchema,
  userSchema,
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

const account = {
  status: "Active",
  name: "Dana Whitfield",
  username: "dwhitfield",
  emailAddress: "dana@example.com",
  timezone: "America/Chicago",
  timeFormat: "24-hour",
  locale: "en",
} as const;

describe("userCreateSchema", () => {
  it("takes the roles as a list of role ids", () => {
    const parsed = userCreateSchema.safeParse({ ...account, roleIds: ["rol_1", "rol_2"] });

    expect(parsed.success).toBe(true);
    expect(parsed.data?.roleIds).toEqual(["rol_1", "rol_2"]);
  });

  it("accepts a user created with no roles", () => {
    expect(userCreateSchema.safeParse({ ...account, roleIds: [] }).success).toBe(true);
  });

  it("refuses role assignment objects where ids belong", () => {
    const parsed = userCreateSchema.safeParse({
      ...account,
      roleIds: [{ roleId: "rol_1", userId: "usr_1" }],
    });

    expect(parsed.success).toBe(false);
  });

  it("has no password field to fill in", () => {
    const parsed = userCreateSchema.parse({ ...account, roleIds: [], password: "hunter22" });

    expect(parsed).not.toHaveProperty("password");
    expect(parsed).not.toHaveProperty("mustChangePassword");
  });

  it("keeps the base schema's rules", () => {
    const parsed = userCreateSchema.safeParse({
      ...account,
      emailAddress: "not-an-address",
      roleIds: [],
    });

    expect(parsed.success).toBe(false);
    expect(parsed.error?.issues.map((issue) => issue.path.join("."))).toContain("emailAddress");
  });
});

describe("userEditSchema", () => {
  const editable = {
    ...account,
    id: "usr_1",
    version: 3,
    isLocked: false,
    mustChangePassword: false,
    jobTitle: "",
    phoneNumber: "",
    createdAt: 1_760_000_000,
    updatedAt: 1_760_000_100,
  };

  it("accepts a user who has never signed in or set a password", () => {
    const parsed = userEditSchema.safeParse({
      ...editable,
      lastLoginAt: null,
      passwordChangedAt: null,
    });

    expect(parsed.success).toBe(true);
    expect(parsed.data?.lastLoginAt).toBeNull();
    expect(parsed.data?.passwordChangedAt).toBeNull();
  });

  it("accepts the timestamps once they exist", () => {
    const parsed = userEditSchema.safeParse({
      ...editable,
      lastLoginAt: 1_760_000_200,
      passwordChangedAt: 1_760_000_150,
    });

    expect(parsed.success).toBe(true);
  });

  it("keeps false toggles as false", () => {
    const parsed = userEditSchema.parse({ ...editable, isLocked: false, mustChangePassword: false });

    expect(parsed.isLocked).toBe(false);
    expect(parsed.mustChangePassword).toBe(false);
  });

  it("does not carry fields the edit form never shows", () => {
    const parsed = userEditSchema.parse({
      ...editable,
      avatarStyle: "dusk",
      availabilityStatus: "away",
    });

    expect(parsed).not.toHaveProperty("avatarStyle");
    expect(parsed).not.toHaveProperty("availabilityStatus");
  });
});

describe("userSchema", () => {
  it("accepts a null last sign-in", () => {
    const parsed = userSchema.safeParse({ ...account, lastLoginAt: null });

    expect(parsed.success).toBe(true);
  });

  it("shows the assistant for an account that has never chosen", () => {
    expect(userSchema.parse(account).assistantHidden).toBe(false);
  });

  it("keeps an assistant the person turned off hidden", () => {
    expect(userSchema.parse({ ...account, assistantHidden: true }).assistantHidden).toBe(true);
  });
});
