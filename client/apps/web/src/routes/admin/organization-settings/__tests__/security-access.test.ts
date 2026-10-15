import { describe, expect, it } from "vitest";
import { securityTabValues } from "@/hooks/use-organization-setting-state";
import { Operation, Resource, type OperationType } from "@trenova/shared/types/permission";
import { firstAllowed, securityAccess } from "../_components/security-access/use-security-access";

function grants(entries: Array<[string, OperationType[]]>) {
  const held = new Map(entries);
  return (resource: string, operation: OperationType) =>
    held.get(resource)?.includes(operation) ?? false;
}

describe("securityAccess", () => {
  it("shows a person who reads only auth events the activity tab and nothing else", () => {
    const access = securityAccess(grants([[Resource.AuthEvent, [Operation.Read]]]));

    expect(access.tabs).toEqual({
      "sign-in": false,
      provisioning: false,
      policies: false,
      activity: true,
    });
    expect(access.activity).toEqual({ auth: true, risk: false, identities: false, mfa: false });
  });

  it("offers the provisioning tab on the audit grant alone", () => {
    const access = securityAccess(grants([[Resource.ProvisioningAudit, [Operation.Read]]]));

    expect(access.tabs.provisioning).toBe(true);
    expect(access.scimDirectories.read).toBe(false);
  });

  it("holds each write control to its own operation", () => {
    const access = securityAccess(
      grants([
        [Resource.IdentityProvider, [Operation.Read, Operation.Update]],
        [Resource.AccessPolicy, [Operation.Read, Operation.Delete]],
      ]),
    );

    expect(access.identityProviders).toEqual({
      read: true,
      create: false,
      update: true,
      remove: false,
    });
    expect(access.accessPolicies).toEqual({
      read: true,
      create: false,
      update: false,
      remove: true,
    });
    expect(access.scimDirectories).toEqual({
      read: false,
      create: false,
      update: false,
      remove: false,
    });
  });

  it("no longer opens anything on an organization grant", () => {
    const access = securityAccess(
      grants([[Resource.Organization, [Operation.Read, Operation.Update]]]),
    );

    expect(Object.values(access.tabs).some(Boolean)).toBe(false);
  });
});

describe("firstAllowed", () => {
  const allowed = {
    "sign-in": false,
    provisioning: false,
    policies: true,
    activity: true,
  } as const;

  it("keeps the requested tab when it is allowed", () => {
    expect(firstAllowed("activity", securityTabValues, allowed)).toBe("activity");
  });

  it("falls back to the first allowed tab in order", () => {
    expect(firstAllowed("sign-in", securityTabValues, allowed)).toBe("policies");
  });

  it("returns null when nothing is allowed", () => {
    expect(
      firstAllowed("sign-in", securityTabValues, {
        "sign-in": false,
        provisioning: false,
        policies: false,
        activity: false,
      }),
    ).toBeNull();
  });
});
