import { describe, expect, it } from "vitest";
import type { User } from "../../types/user";
import { keepOrganizationSetup } from "../auth-store";

function user(overrides: Partial<User> = {}): User {
  return {
    id: "usr_01",
    currentOrganizationId: "org_01",
    businessUnitId: "bu_01",
    name: "Dana Whitfield",
    username: "dana",
    emailAddress: "dana@acme.example",
    timezone: "America/New_York",
    ...overrides,
  } as User;
}

const incomplete = { complete: false, missing: ["addressLine1"], canComplete: true };

describe("keepOrganizationSetup", () => {
  it("keeps the last answer when a self write returns the user without one", () => {
    const next = keepOrganizationSetup(
      user({ organizationSetup: null, mustChangePassword: false }),
      user({ organizationSetup: incomplete, mustChangePassword: true }),
    );

    expect(next?.organizationSetup).toEqual(incomplete);
    expect(next?.mustChangePassword).toBe(false);
  });

  it("takes a fresh answer over the last one", () => {
    const complete = { complete: true, missing: [], canComplete: false };

    const next = keepOrganizationSetup(
      user({ organizationSetup: complete }),
      user({ organizationSetup: incomplete }),
    );

    expect(next?.organizationSetup).toEqual(complete);
  });

  it("drops the answer of another organization", () => {
    const next = keepOrganizationSetup(
      user({ currentOrganizationId: "org_02", organizationSetup: null }),
      user({ organizationSetup: incomplete }),
    );

    expect(next?.organizationSetup).toBeNull();
  });

  it("signs out as nobody", () => {
    expect(keepOrganizationSetup(null, user({ organizationSetup: incomplete }))).toBeNull();
  });
});
