import { describe, expect, it } from "vitest";
import {
  authEventSchema,
  provisioningAuditRecordSchema,
  riskDecisionSchema,
} from "@trenova/shared/types/iam";

/**
 * These three records exist to describe what happened when something went
 * wrong, so the columns naming who it happened to are the ones most likely to
 * be empty.
 *
 * `AuthEvent.UserID`, `RiskDecision.UserID` and their tenant columns
 * (services/tms/internal/core/domain/iam/models.go) are `pulid.ID` declared
 * without `notnull`, on purpose: a login rejected before the account is
 * identified has no user, and no organization either.
 * `ProvisioningAuditRecord.ResourceID` is the same — a provisioning action that
 * failed never created a resource to point at.
 *
 * `pulid.ID.MarshalJSON` writes `null` for a nil id, so those are exactly the
 * rows that arrive with nulls. `z.string().optional()` does not admit null, and
 * one such row discards the whole response through `safeParse`.
 *
 * The neighbouring text columns go the other way: `ipAddress`, `userAgent`,
 * `mfaState`, `errorCode`, `reason` and `externalId` are Go `string`, so an
 * unset one is always `""`.
 */
describe("authEventSchema", () => {
  const preAuthFailure = {
    id: "aue_01JAUTHEVENT000000000000",
    userId: null,
    organizationId: null,
    businessUnitId: null,
    identityProviderId: null,
    provider: "password",
    outcome: "failed",
    ipAddress: "203.0.113.10",
    userAgent: "",
    authenticatorAal: 1,
    federationFal: 1,
    mfaState: "",
    riskOutcome: "deny",
    riskSignals: null,
    errorCode: "unknown_user",
    occurredAt: 1_702_592_000,
    createdAt: 1_702_592_000,
  };

  it("reads a login rejected before any account was identified", () => {
    const parsed = authEventSchema.parse(preAuthFailure);

    expect(parsed.userId).toBe("");
    expect(parsed.organizationId).toBe("");
    expect(parsed.businessUnitId).toBe("");
    expect(parsed.identityProviderId).toBe("");
    expect(parsed.riskSignals).toEqual([]);
  });

  it("keeps the ids of an event that names its user", () => {
    const parsed = authEventSchema.parse({
      ...preAuthFailure,
      userId: "usr_01JUSER000000000000000AB",
      organizationId: "org_01JORGANIZATION00000000",
      outcome: "success",
      riskOutcome: "allow",
      errorCode: "",
    });

    expect(parsed.userId).toBe("usr_01JUSER000000000000000AB");
    expect(parsed.organizationId).toBe("org_01JORGANIZATION00000000");
  });
});

describe("riskDecisionSchema", () => {
  it("reads a decision made before the user was known", () => {
    const parsed = riskDecisionSchema.parse({
      id: "rd_01JRISKDECISION000000000",
      userId: null,
      organizationId: null,
      businessUnitId: null,
      outcome: "deny",
      signals: ["impossible_travel"],
      reason: "",
      createdAt: 1_702_592_000,
    });

    expect(parsed.userId).toBe("");
    expect(parsed.organizationId).toBe("");
    expect(parsed.signals).toEqual(["impossible_travel"]);
  });
});

describe("provisioningAuditRecordSchema", () => {
  it("reads an action that failed before it created anything", () => {
    const parsed = provisioningAuditRecordSchema.parse({
      id: "par_01JPROVISIONAUDIT0000000",
      organizationId: "org_01JORGANIZATION00000000",
      directoryId: "scd_01JSCIMDIRECTORY00000000",
      action: "create",
      resourceType: "user",
      externalId: "okta-90210",
      resourceId: null,
      status: "failed",
      errorMessage: "Email already in use.",
      createdAt: 1_702_592_000,
    });

    expect(parsed.resourceId).toBe("");
    expect(parsed.externalId).toBe("okta-90210");
  });
});
